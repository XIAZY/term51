### `stream_readonly.zig`

File: `terminal/stream_readonly.go`. The wireframe header (names, the
`Stream[*StreamReadonlyHandler]` instantiation, `Vt(action, value)`) is
followed as written; the notes below are what the implementation adds.

- `vt`'s `switch (action)` over the comptime tag is a Go `switch action`;
  each prong reads the payload field of `value` named by the tag
  (`value.CursorUp.value`, `value.SetMode.mode`, ...). Zig's switch is
  exhaustive over `Action.Tag`; the Go switch lists every key (the empty
  prongs for DCS, APC and the "no terminal-modifying effect" group keep
  their comments) and has `default: panic("unreachable")`.
- `try f()` is `if err := f(); err != nil { return err }`; nothing follows
  the switch, so the function ends with `return nil`. `setAttribute(value)
  catch {}` is `_ = self.terminal.SetAttribute(...)`.
- Integer widths: the payloads are u16 (cursor movement, margins) or usize
  (counts); Terminal's functions take `usize`, so every u16 is widened with
  `uint(...)`. `cursor.y + 1` / `cursor.x + 1` are u16 additions that cannot
  overflow (y < rows, x < cols, both <= maxInt(u16)), so they are plain
  `+ 1` on SizeCellCountInt. `cursor.x + 1 +| value.value` is
  `(x + 1) +| value` (same precedence, left to right): `saturatingAddU16(x+1,
  v)` (zig.go).
- `.kitty_keyboard_pop => ...pop(@intCast(value))`: u16 to usize always
  fits; `uint(value.KittyKeyboardPop)`.
- `.cursor_style`: the two value-producing switches are switch statements
  that assign `blink` and `style`; the enum is exhaustive in Zig, so each
  has `default: panic("unreachable")`.
- `.modify_key_format`'s inner `switch (value) { .other_keys_numeric => ...,
  else => {} }` is kept as a switch with an empty default.
- `inline fn horizontalTab` / `horizontalTabBack`: ordinary methods;
  `for (0..count) |_|` is `for range count` (count is evaluated once in both).
- setMode `.@"132_column" => try self.terminal.deccolm(self.terminal.screens.active.alloc, ...)`:
  Terminal.Deccolm has no allocator parameter (Terminal.go), so the
  allocator argument is dropped; the `if (enabled) .@"132_cols" else
  .@"80_cols"` is an if/else that assigns a local.
- setMode `.enable_left_and_right_margin`: `self.terminal.cols - 1` is a u16
  subtraction; cols > 0 is guaranteed by Terminal.init, so it is a plain
  `- 1`.
- colorOperation: `requests.constIterator(0)` / `it.next()` are the
  OSCParsersColorList stand-ins (osc_parsers_color.go); `while (it.next())
  |req|` is `for req := it.next(); req != nil; req = it.next()`. The tagged
  union switches (`.set => |set|`, `.palette => |i|`, ...) switch on `Tag`
  and read the payload field; exhaustive enum switches get `default:
  panic("unreachable")`, Zig's `else => {}` prongs an empty `default`.
- `.reset_palette`: `mask.iterator(.{})` is `StaticBitSet256.iterator()`
  (color.go) and `@intCast(i)` (usize to u8) is `uint8(i.Value)` (i < 256).
  The loop resets bits while iterating (`palette.reset` unsets bit i). Zig's
  BitSetIterator copies the current word; the Go iterator reads the set at
  each step. The bits it unsets are ones already passed, so both visit the
  same indexes. `mask.* = .initEmpty()` is `*mask = StaticBitSet256InitEmpty()`.
- kittyColorOperation: `for (request.list.items) |item|` ranges over the
  `list` slice (kitty_color.go).
- colorOperation and kittyColorOperation have empty inferred error sets; the
  Go result is always nil (wireframe).

### `render.zig`

File: `terminal/render.go`. The wireframe header (Go-memory rule: row_data
and cells are slices, the per-row arena is dropped, allocator parameters
dropped) is followed as written; notes on what the implementation adds:

- `RenderState.empty` is a package var initialised at start-up with the
  three non-zero fields (palette ColorDefault, visual_style
  CursorStyleBlock, visible true); `cursor.style` (Zig: undefined) is the
  zero Style.
- `Dirty.String()` reads a names array (`renderStateDirtyNames`).
- Packed-struct backing integers: `@bitCast(t.flags.dirty) > 0` and
  `@bitCast(t.screens.active.dirty) > 0` are `!= (TerminalDirty{})` /
  `!= (ScreenDirty{})`: any flag set. `t.flags.dirty = .{}` / `s.dirty =
  .{}` assign the zero struct (every field default false).
- Labeled blocks: `redraw:` and `bg_fg:` (value / early exit), `dirty:`
  (left with `break :dirty`, ends with `continue`), `selection:`, `cache:`
  and `cache_check:` are Go `{ }` blocks left with `goto <label>_done` (or
  `goto dirty`) to a label right after the block (page.go/formatter.go
  precedent); each block is its own Go block so no goto jumps over a
  declaration. `break :selection` from inside `cache:` is `goto
  selection_done`, a label at the end of the `if s.selection.Valid` body.
  `bg_fg:` evaluates `foreground.get()` only after `background.get()` was
  non-null, as Zig does.
- `while (row_it.next()) |row_pin| : (y = y + 1)`: a Go for whose post
  statement is `row_pin_, y = row_it.Next(), renderAddOneU16(y)`; the
  continue expression runs on `continue` as in Zig. `y + 1` on u16 traps in
  Zig: `renderAddOneU16` panics on overflow (unreachable: rows <=
  maxInt(u16)).
- `std.MultiArrayList.resize` (rows and cells) is `renderResize`: within
  the capacity the slice is resliced and the items between the old and new
  length keep what they held (Zig: undefined, and the old bytes are what is
  in memory); beyond the capacity a new slice is made and the old items
  copied, the rest Go zero. `shrinkRetainingCapacity(rows)` is
  `row_data[:rows]` after setting the dropped rows' cells to nil (Zig deinits
  them). `clearRetainingCapacity()` is `cells[:0]`. The shrink keeps the
  capacity, which the test "rows and cells are reused across resizes"
  checks.
- `row_highlights[y] = .empty` is `nil`; `arena.reset(.retain_capacity)`,
  `row_arenas[y].promote`/`defer ... = arena.state` and the `Row.arena`
  field are dropped (Go-memory rule).
- `fastmem.copy(page.Cell, cells_slice.items(.raw), page_cells)` is a loop
  that sets `cells[x].raw = page_cells[x]` (the struct-of-arrays copy).
- `arena_alloc.dupe(u21, p.lookupGrapheme(page_cell) orelse &.{})` is a
  fresh `make([]uint32, n)` + `copy` (an empty, non-nil slice when there is
  no grapheme).
- `p.styles.get(p.memory, id).*` is `*p.styles.Get(id)`;
  `hyperlink_set.get(memory, id)` is `hyperlink_set.Get(id)`; `link.eql(
  link_page.memory, other, other_page.memory)` is `link.Eql(other)`
  (HyperlinkPageEntry.Eql has no memory parameters).
- `pointFromPin(.screen, pin).?.screen` asserts Valid and reads `.Screen`.
- `updateHighlightsFlattened`: `hl.chunks.slice().items(.node/.start/.end)`
  are the `chunks[i]` fields; `ends[nodes.len - 1] - 1` and `self.cols - 1`
  are u16 subtractions that cannot trap on the path that reaches them (the
  chunk's end is above row_pin.y; cols > 0 when there are rows), so they
  are plain `- 1`. `row_highlights.append(arena_alloc, ...)` is Go append.
- `string`: `std.unicode.utf8CodepointSequenceLength` is ported as
  `stdUnicodeUtf8CodepointSequenceLength` in the new file
  `terminal/std_unicode.go` (Zig 0.15.2 std/unicode.zig; new sentinel
  `ErrCodepointTooLarge`). Go's `utf8.RuneLen` is not used because it
  returns -1 for surrogate halves where Zig returns 3. std has no test
  block for this function, so there is no matching test file.
  `catch return error.WriteFailed` returns ErrWriteFailed; `print("{u}",
  cp)` is `oscWriter.printUnicodeCodepoint`; `m.map.appendNTimes(alloc, c,
  len)` reuses `formatterAppendNTimes` (formatter.go). `@intCast(x)` (to u16)
  and `@intCast(y)` (to u32) always fit (x < cols, y < rows).
- `CellSet` (std.AutoArrayHashMapUnmanaged(Coordinate, void)): the
  wireframe's insertion-ordered key slice; `Put` appends a key not already
  present (linear `Contains`), `Count` is the length, `Deinit` sets the
  slice to nil. Same keys in the same order as std.
- `linkCells`: `link:` blocks are Go `{ }` blocks; `orelse return result`
  / `orelse continue` inside them return or continue directly (the inner
  `continue` continues the x loop, as in Zig). `errdefer result.deinit`
  is memory only. `viewport_point.y >= row_pins.len` compares as uint.
- Deinit sets each row's cells to nil and row_data to nil (Zig's deinit
  leaves them undefined; Go zero value).
