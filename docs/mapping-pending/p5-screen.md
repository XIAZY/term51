### `Screen.zig`

File: `terminal/Screen.go`. The wireframe header (Go-memory rule, widths,
Zig defaults, omissions) is followed as written; no signature changed. The
notes below are what the implementation adds. Idioms are PageList.go's
(named `err` result + deferred closure for an `errdefer` that changes state,
`.?` = `assert(x.Valid)` / `assert(p != nil)`, `log.err` =
`warn("screen", ...) // Zig: log.err`).

#### Pointers and memory

| Zig | Go | Notes |
|---|---|---|
| `@as([*]Cell, @ptrCast(self.cursor.page_cell)) + n` / `- n` (cursorCellRight/Left, cursorRight/Left) | `&page.cells[page.cellIndex(self.cursor.page_cell) ± n]`, `page` the cursor pin's page | cellIndex (page.go) panics if `page_cell` is not in the cursor page; Zig would step an unchecked pointer. Upstream finding 15.6 (page_row/page_cell left in the old page after an OutOfSpace split) would panic in Go if a later call stepped that stale pointer; no test reaches it |
| `cell_current - self.cursor.x` then `@memset(cells[0..cols], blank)` (cursorDownScroll, cursorScrollAbove) | `start := cellIndex(page_cell) - x; cells := page.cells[start : start+cols]`; loop assignment | |
| appendGrapheme's `cell_idx` (`(@intFromPtr(target) - @intFromPtr(zero)) / @sizeOf(Cell)`) | `cellIndex(cell) - (cellIndex(page_cell) - x)` with `assert(target >= zero)` (Zig's usize `-` traps) | |
| clearCells' runtime-safety address checks (row within `rows[0..size.rows-1]`, first/last cell within the row's cells) | `page.rowIndex(row) <= size.rows-1`; `cellIndex(&cells[0]) >= cellIndex(&row_cells[0])`, same for the last | rowIndex/cellIndex panic for a pointer of another page, where Zig compares raw addresses (and would usually also fail). Go-only panic tests rely on it |
| `fastmem.rotateOnceR(Row, s)` | `screenRotateOnceR` (not a Zig declaration): `tmp := s[len-1]; copy(s[1:], s[:len-1]); s[0] = tmp` | an empty slice panics as Zig's `len - 1` traps |
| `page.rows.ptr(page.memory.ptr)` then `rows[a..b]` | `page.rows[a:b]` | bounds-checked against capacity (as PageList.go) |
| `row.cells.ptr(chunk.node.data.memory)[0..cols]` (clearRows) | `chunk.node.data.rowCells(row)[0:cols]` | |
| `const page = self.cursor.page_pin.node.data;` in testWriteString (a COPY of the Page struct) then `page.styles.use(...)` | the same copy in Go | `use` only writes the item's ref count in the shared items slice, so the copy behaves as in Zig |
| `self.alloc.create(Hyperlink)` / `destroy` | `new(Hyperlink)` / comment | `Hyperlink.deinit` is still called where Zig calls it |
| `PageList.Clone.TrackedPinsRemap.get(p)` | `pin_remap[p]` (nil when absent) | |
| `std.ArrayList(Pin)` in selectionString, `pins.toOwnedSlice` | `[]PageListPin` behind `FormatterPinMap.map_`; toOwnedSlice copies exactly `len` items | |
| `aw.toOwnedSliceSentinel(0)`, `alloc.dupeZ(u8, text)` | method `oscWriterAllocating.toOwnedSliceSentinel` in Screen.go (std 0.15.2's Allocating has it; osc.go had only toOwnedSlice); dupeZ written inline | both follow the `[:0]const u8` convention of StringMap.go (len = text, the 0 at index len inside cap). The method could move to osc.go |

#### Control flow

- Labeled blocks with early exits (`cursor:`/`remap:` and `sel:` in clone,
  `start_pin:`/`end_pin:` in selectLine, `end:`/`start:` in selectWord,
  `prompt:` in selectOutput and resize, `start_x:` in promptClickLine) are
  closures; a `return` in a Zig block that returns from the enclosing
  function is reported by the closure (`end_pin` returns an Optional,
  selectOutput's closure returns a `done` flag). Blocks with only a final
  `break :x value` are plain Go blocks or if/else (`start:`/`end:` in
  selectLine and selectAll use a `found` flag; `new_pin:` in
  cursorDownScroll; `id:` and `space:` in manualStyleUpdate).
- `while (it.next()) |p| { ... } else { ... }` (selectLine's start_pin) is
  the loop followed by the else body; every exit from the loop body is a
  `return`, so the else only runs when the iterator is exhausted.
- `while (x0 < cells.len) clear: { ... break :clear ... }`
  (clearUnprotectedCells): `break :clear` leaves the loop body, so it is
  `continue` (the condition then ends the loop).
- `row_it:` labeled loops in promptClickLine are Go labeled `break`s
  (`right_row_it`, `left_row_it`: two labels because Go labels are
  function-scoped).
- `errdefer` with an effect: `pages.deinit()` in init and clone (it asserts
  integrity), `self.cursor = old` in cursorCopy, the implicit-id rollback in
  startHyperlink, the style restore in setAttribute: deferred closures on
  the named `err`. `errdefer`s that only free, or after which nothing
  fails (`untrackPin` in init, `destroy`/`deinit` of the link in
  startHyperlinkOnce, `tracked_sel.deinit` in select, `styles.release` in
  manualStyleUpdate), are comments.
- `defer self.cursor.page_pin.node.data.assertIntegrity()`
  (appendGrapheme): Zig evaluates the whole expression when the defer runs;
  Go would evaluate the receiver at the `defer` statement, so it is a
  closure (the cursor page changes in between when capacity grows).
- Zig's `defer`s in resize (assertIntegrity, the style restore, the saved
  cursor untrack) are Go defers registered in the same order.
- `defer self.cursor.hyperlink_implicit_id += 1` inside the `implicit:`
  block of startHyperlink runs at the end of that block: the Go code takes
  the id and increments right after.
- Error switches (`catch |err| switch (err)`) are `switch err` with
  `default: panic("unreachable")`; `catch unreachable` is
  `panic("unreachable")` on a non-nil error.
- The `std.math.order` switch in appendGrapheme is a `switch {}` on `==`,
  `<`, default.

#### Integer traps made explicit

`self.cursor.x + n < cols` / `self.cursor.y + n < rows` (u16 in Zig, traps
on overflow) are computed in uint32 so an overflow fails the assertion
instead of wrapping (cursorCellRight, cursorRight, cursorDown). `@intCast`
of an active `y` (u32) to u16 is asserted (clone, cursorReload, resize's
saved cursor), as is `cell_idx - x` to u16 in appendGrapheme.
`hyperlink_implicit_id += 1` (u32) is asserted not to overflow (upstream
finding 15.7). `size.rows - 1` / `cols - 1` where the operand is not bounded
by an earlier check are asserted (`clone`'s padded end pin,
cursorScrollAboveRotate, clearCells). selectLine's `x - 1` (usize) is
asserted.

#### Deviation (decided before the port)

`clone` returns `ErrScreenCloneCursorOutside` (new sentinel, Screen.go) where
upstream aborts in the final `assertIntegrity` (cursor outside the cloned
region and the clone has more rows than the active area). The check is the
exact abort condition (`pointFromPin(.active, cursor pin)` is null), placed
just before `assertIntegrity`; the errdefer then deinits the clone. No Go
test exists for it yet (term51 docs say there is one): a Go-only test should
be added (Screen_2_test.go is read-only for the porter).

#### `testWriteString`

- `std.unicode.Utf8View.init` + iterator: `utf8.Valid` then
  `utf8.DecodeRune`. Zig 0.15.2's `utf8ValidateSlice` rejects overlong
  forms, surrogate halves and code points above U+10FFFF, as `utf8.Valid`
  does, so the accepted set is the same and the iterator only sees valid
  input. (Not wrapped under a Zig name: it is one use in a test helper; the
  lead may prefer a `stdUnicodeUtf8View` wrapper.)
- `unicode.table.get(c).width` for `c > 0xFF`: `UnicodeTable.Get(c).width`
  (unicode_props_table.go), converted to `uint` (Zig's `@intCast` of the u2
  to usize cannot fail).

#### `std.mem.indexOfAny` (`terminal/std_mem.go`)

`stdMemIndexOfAny` / `stdMemIndexOfAnyPos`, ported from Zig 0.15.2
`lib/std/mem.zig` (generic over `comparable`). Go's `slices.Index` takes one
value, not a set. std has no test block for either function, so there is no
test file.

#### Notes for the formatter porter

`selectionString` and `dumpString` build `FormatterOptions` with
`codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true}` (Zig's
default `.{}`, empty and non-null) and set `content` to
`.selection = Some(sel)`.
