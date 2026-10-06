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
  (rows for docs/go-only-tests.md: below).

## Page-layer bookkeeping: cursor row pointers after a split that moves the cursor row (ruled departure)

Upstream-findings section 15, Screen items 6 and 33; term51 case
`text/style_split/split_moves_cursor_row_erase` (`upstream_invalid`, stays
excluded). After `splitForCapacity` moves the cursor row to a new page,
upstream leaves `cursor.page_row`/`page_cell` pointing at the old page's row,
which the split cleared and hid. Its release build prints into that hidden
row, then (after CR or a same-row CUP, which reload only `page_cell`) leaves a
styled cell in a visible row not marked styled: a state its own
`verifyIntegrity` rejects (`UnmarkedStyleRow`); its Debug build aborts on the
next page check. Ruling (project lead): same class as `Screen.clone` — where
upstream's release build ends in a state its own integrity check rejects, the
port does the evidently intended thing (or returns an error) rather than
reproduce the corrupted state; a ruled departure, not a fix (term51
docs/upstream-crashes.md, aborts and invalid-states list; deliberate-fixes.md
entry 8).

- `splitForCapacity` (Screen.go), after `cursorChangePin(new_cursor)`, sets
  `page_row`/`page_cell` from `cursor.page_pin.RowAndCell()`, exactly the lines
  `cursorAbsolute` runs after its `cursorChangePin`. Not `cursorReload`: x and y
  are unchanged by the split, and `cursorReload` would also re-run
  `manualStyleUpdate` (the caller of `splitForCapacity`) for a non-default style.
  Before: Go panicked at the first print after the split ("cell is not in the
  page", `Screen.CursorRight`).
- Go-only test: `TestScreenSplitRuling_split_that_moves_the_cursor_row_prints_into_the_visible_row`
  (the corpus case's steps: the text lands in the visible row with the new
  style, the row is marked styled, every page passes `VerifyIntegrity` and the
  page list `verifyIntegrity`, also after the erase that makes upstream's Debug
  build abort). Checked to fail without the change.

## Rows ready for docs/go-only-tests.md

| Screen_split_rulings_test.go | TestScreenSplitRuling_full_style_set_in_the_rows_above_fails_the_style_change | ruling (Screen.splitForCapacity returns ErrOutOfSpace where upstream aborts in Page.exactRowCapacity's style-capacity cast; docs/mapping.md "Page-layer bookkeeping") |
| Screen_split_rulings_test.go | TestScreenSplitRuling_full_style_set_in_the_rows_below_fails_the_style_change | ruling (as above, the range below the cursor) |
| Screen_split_rulings_test.go | TestScreenSplitRuling_split_that_moves_the_cursor_row_prints_into_the_visible_row | ruling (Screen.splitForCapacity reloads cursor.page_row/page_cell when the split moves the cursor row to a new page; upstream ends in a state its integrity check rejects; docs/mapping.md "Page-layer bookkeeping") |

## Text ready for docs/mapping.md ("Page-layer bookkeeping", next to the clone deviation)

`splitForCapacity` returns `ErrOutOfSpace` where upstream aborts in
`exactRowCapacity`'s `.styles = @intCast(styles_cap)` (a row range referencing
every id of a full style set of the maximum capacity; helper
`exactRowCapacityOrError`, not a Zig declaration; `ExactRowCapacity` itself
still asserts, and so does `PageList.Compact`'s call, unreachable from the
library). When the split moves the cursor row to a new page,
`splitForCapacity` reloads `cursor.page_row`/`page_cell` from the moved pin
(upstream leaves them in the old page and ends in a state its integrity check
rejects). Both are project-lead rulings; Go-only tests in
`Screen_split_rulings_test.go`.
