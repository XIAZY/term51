# Mapping notes pending merge: `page.zig` (worktree p4-page)

Sections ready to fold into `docs/mapping.md` under the page-layer module.

### `page.zig`

File: `terminal/page.go`.

#### Memory

| Zig | Go | Notes |
|---|---|---|
| `Page.memory`, `Offset(Row)`, `Offset(Cell)` | `rows []PageRow`, `cells []PageCell` (one slice, `capacity.rows * capacity.cols`) | wireframe decision; `Row.cells` keeps the byte offset `cells_start + index*8` |
| `row.cells.ptr(self.memory)` (many-item pointer to the row's first cell) | `self.rowCells(row)`: `self.cells[(row.cells - cells_start)/8:]`, sliced by the caller as Zig slices the pointer | private helper, not a Zig declaration. A row of another page whose offset is out of range panics when sliced |
| `cells_start` outside `layout` | `pageCellsStart(cap)`: `alignForward(cap.rows * @sizeOf(Row), @alignOf(Cell))`, the first lines of `layout` | computing the whole layout per call would be slow (it runs the set layouts' float formulas) |
| `getOffset(Cell, self.memory, cell)` | `self.cellOffset(cell)` = `pageCellsStart + cellIndex(cell)*8`; asserts the result fits u32 (`@intCast` to OffsetInt) | |
| pointer subtraction (`cellIndex`, `rowIndex`) | `unsafe` pointer arithmetic, the one sanctioned use (wireframe): panics if the pointer is not an element of the page's slice (also for an empty slice) | Go-only panic tests `GetCells` of another page's row rely on it |
| `initBuf(buf, l)` | private `pageInitBuf(rows, cells, l)`: the buffer is the rows and cells slices; the sub-structures make their own memory | the wireframe omitted initBuf; it is reinstated (private) because `init` and `reinit` both call it and `reinit` must reuse the page's own rows and cells: Zig zeroes the same memory, so a `*Row`/`*Cell` held elsewhere (for example Screen's cursor during `PageList.eraseRows`' reinit of the last page) still points into the page. A fresh allocation would leave such pointers on a detached copy. `PageList`'s prune path (`first.data = .initBuf(.init(buf), layout)`) can use it the same way |
| `init`: `posix.mmap` + `errdefer munmap` | `make` of rows and cells; the error result is always nil | `assert(l.total_size % page_size_min == 0)` kept |
| `deinit`: `munmap`, `self.* = undefined` | `*self = Page{}` | as `Tabstops.deinit` |
| `reinit`: `@memset` of the memory as u64, then `initBuf` | `clear(rows)`, `clear(cells)`, `pageInitBuf(self.rows, self.cells, layout(capacity))` | |
| `clone` / `cloneBuf` (mmap + memcpy of the whole page) | `Clone`: copy of the struct, then of every slice the page owns: rows, cells, both allocators (bitmaps and chunks), both hash maps (metadata, a new header, keys, values) and both sets (table, items). Private helpers `pageCloneBitmapAllocator`, `pageCloneHashMap`, `pageCloneRefCountedSet` | the copies reach into the private fields of bitmap_allocator.go, hash_map.go and ref_counted_set.go (same package) |
| an `Offset(T).Slice` copied with the memory (grapheme map values, hyperlink entries' uri and explicit id) | `pageRebase(s, from, to, chunk_len)`: finds `&from[i] == &s[0]` among the chunk starts of the old allocator and returns `to[i : i+len(s) : i+cap(s)]` | no `unsafe` (not sanctioned here), so a linear scan of chunk starts per slice (graphemes: up to 512 starts for 8192 bytes). Capacity kept (appendGrapheme grows into it). An empty slice, or one not inside the allocator (not page memory), is returned unchanged |
| `fastmem.copy(Cell, dst, src)` | `copy(dst, src)` | Go's copy is a memmove; upstream's memcpy with overlapping ranges is undefined (it is never reached with overlap in the library's own calls as far as checked) |
| `@memset(@as([]u64, @ptrCast(cells)), 0)` | `clear(cells)` | the zero PageCell is the all-zero cell |

#### Byte arithmetic (`layout`, `Capacity`)

- All byte sizes and offsets are `uint64` (wireframe). The sizes and base
  alignments Zig reads from types are untyped constants next to the
  wireframe's `...Sizes` values: `pageSizeOfRow`/`pageSizeOfCell`/
  `pageAlignOfCell` 8, `pageBitSizeOfRow`/`pageBitSizeOfCell` 64,
  `pagePageSizeMin` 4096, and the base alignments:
  StyleSet 8 (RefCountedSet's `@max` includes `@alignOf(Layout)`, which has
  usize fields), GraphemeAlloc and StringAlloc 8 (`@alignOf(u64)`),
  GraphemeMap 8 (`@max(Header 4, key 4, value 8)`), hyperlink.Set 8,
  hyperlink.Map 4 (`@max(4, 4, 2)`). The wireframes of the three
  sub-structure files omit `base_align`; page.go needs the numbers for the
  `alignForward`/`alignBackward` calls.
- `std.mem.alignForward/alignBackward(usize, ...)` are `pageAlignForward` /
  `pageAlignBackward` (Zig 0.15.2: assert the alignment is a power of two;
  alignForward's `addr + (alignment - 1)` asserted not to overflow).
  `std.math.divCeil(usize, ...)` is `pageStdMathDivCeil` (unsigned branch;
  every caller's `catch` is for a zero divisor, which never happens).
  `std.math.ceilPowerOfTwo(T, v) catch unreachable` and
  `ceilPowerOfTwoAssert(u32, v)` are `pageStdMathCeilPowerOfTwo(v, bits)`
  (asserts `v != 0` and that the result fits T). They could move to zig.go
  and be shared with ref_counted_set.go's inline versions.
- `layout`: `hyperlink_count` is a u16 (`@divFloor` of the u16 field);
  `hyperlink_count * hyperlink_cell_multiplier` is a u16 product of at most
  1365 * 16, so it cannot trap and `std.math.cast(u32, ...) orelse
  maxInt(u32)` never takes its `orelse` (comment in place).
  `@intCast(grapheme_count)` to the map's u32 Size is asserted.
- `availableBitsForGrid`: the usize subtractions cannot underflow (each
  backward start is at or after the forward start of the same member); no
  assert added, comment in place.
- `adjust`: `@intCast(new_rows)` to u16 is asserted
  (`TestPg1_panic_Capacity_adjust_new_row_count_overflows_u16`).
- `exactRowCapacity`: the four `@intCast`s of the result are asserted
  (`..._styles_capacity_overflows_u16`). `std.StaticBitSet(65536)` is the
  private `pageCellCountSet` (1024 u64 masks, `set`, `isSet`, `count`; a
  u16 index is always in range).

#### Packed structs `Row` and `Cell`

- `PageRow.bitCast` / `pageRowFromBits` and `PageCell.bitCast` /
  `pageCellFromBits` place every field at its Zig bit offset; a field wider
  in Go than in Zig is masked to the Zig width (u2 enums, the u21
  codepoint, the u23 `_padding`), as in style.go.
- `Cell.content` is an untagged packed union of three views of 24 bits; Go
  has one field per view. `pageCellFromBits` fills all three from the same
  bits. `bitCast` takes the view `content_tag` selects for its own bits
  (codepoint 0..20, palette 0..7, rgb 0..23) and the bits above that view
  from `color_rgb`, the one view covering all 24. This reproduces Zig for
  every cell built by a whole literal (other views zero) or read from raw
  bits, including the leftover bits 21..23 under a codepoint tag
  (`term51: Cell.isZero: leftover bits are not ignored`), and for a Go
  write of `content.codepoint` alone (Zig's field write keeps bits 21..23).
  It differs from Zig only for a Go value whose views were set
  inconsistently by hand (no library code does that).
- `Cell.init`: `@bitCast(@as(u64, 0))` is the Go zero value; the u21
  parameter is asserted `<= 0x1FFFFF`.
- Enum `String()`s read names arrays (`pageRowSemanticPromptNames`,
  `pageCellContentTagNames`, `pageCellWideNames`,
  `pageCellSemanticContentNames`); exhaustive switches on these enums
  (`hasText`, `codepoint`, `gridWidth`, `isEmpty`, the `wide` switch of
  `verifyIntegrity`) end in `panic("unreachable")`.

#### Control flow

- `clonePartialRowFrom`: the labeled blocks `hyperlink:` and `dst_id:` are
  left from inside nested blocks, so they are `goto hyperlink` / `goto
  dst_id` with the label after the block (mapping's goto rule); `style:` is
  the last statement of the loop body, so `break :style` is `continue`.
  `copy:` is a plain block assigning `*dst_row`. The deferred
  `pauseIntegrityChecks(false); assertIntegrity()` is a deferred closure;
  Go also runs it while a panic unwinds (Zig aborts first), which only
  matters to the panic tests and does not change that they panic.
- `addWithId... catch |e| switch (e) { ... } orelse id`: an error switch
  over `ErrOutOfMemory` / `ErrNeedsRehash` with `default:
  panic("unreachable")`, then `Valid ? Value : id`. The same shape for
  `string_alloc.alloc ... catch |err| switch (err)` in insertHyperlink.
  Where Zig only asserts the error set at compile time (`comptime
  assert(@TypeOf(e) == error{OutOfMemory})`), any error maps to the page
  error.
- `errdefer` (insertHyperlink, setGraphemes, appendGrapheme): written out
  before the later error returns, in Zig's unwinding order (the explicit
  id, then the uri); errdefers with nothing failing after them are
  comments. As upstream, a failed `addWithIdContext` in
  clonePartialRowFrom leaks the duplicated strings in the page's string
  allocator (no errdefer upstream).
- `moveCells`' parameter `len` hides Go's builtin inside the function;
  `src_cells.len` is written as `len` (the slice was cut to that length).
- `pauseIntegrityChecks`: `+= 1` / `-= 1` on the usize counter are guarded
  by asserts (Zig traps on overflow/underflow;
  `TestPg1_panic_Page_pauseIntegrityChecks_unpause_below_zero`).
- `assertIntegrity`: `log.err` goes through `warn("page", ...)` marked `//
  Zig: log.err` (the stream.go precedent), then `panic("page integrity
  violation")`. The DebugAllocator is dropped.
- `verifyIntegrity`: the Valgrind early return is dropped (comment). The
  arena and the two `std.AutoHashMap`s are Go maps; `getOrPut` + `+= 1` is
  `m[k] += 1`. Go's map order is not Zig's, so when several styles (or
  hyperlinks) have a short ref count, a different one may be logged first;
  the returned error is the same. The zombie-style block is kept with its
  unused counter (`_ = zombies`).
- `self.page.?`-style unwraps (`lookupGrapheme(...).?`, `getEntry(...).?`,
  `getPtr(...).?`) are `assert(x.Valid)` / `assert(p != nil)`.
- `build_options.kitty_graphics` blocks: not ported (wireframe), a comment
  marks each place.

#### Integrity checks are always on

`pageSlowRuntimeSafety` is the constant `true` (wireframe), so every
`assertIntegrity` (clearCells, clearGrapheme, clearHyperlink, setGraphemes,
appendGrapheme, setHyperlink, moveCells, swapCells, cloneFrom,
clonePartialRowFrom) runs a full `verifyIntegrity`, which hashes every cell
of the page into the hyperlink map. The recorded corpus comes from a build
where this is off. See the optimisation list.

#### Tests

- page_1/2/3_test.go: 320 translated Zig tests (all 331 Zig tests are
  mapped; 11 listed in not-translated.txt) plus the Go-only panic tests.
  Results: see the final report of this worktree.
