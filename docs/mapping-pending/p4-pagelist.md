# Mapping notes pending merge: `PageList.zig` (worktree p4-pagelist)

Sections ready to fold into `docs/mapping.md` under the page layer.

### `PageList.zig`

File: `terminal/PageList.go`. The wireframe header (Go-memory rule, pools,
`pageListMemoryLen`, list and pin-set stand-ins, widths) is followed as
written; the notes below are what the implementation adds.

#### Memory and byte accounting

| Zig | Go | Notes |
|---|---|---|
| page buffer from `pool.pages.create()` (pooled) or `page_alloc.alignedAlloc(layout.total_size)` | `make` of the page's rows and cells, then `pageInitBuf(rows, cells, layout)` (page.go) | in `initPages` and `createPageExt` |
| `page_size += page_buf.len` (initPages, createPageExt) | `+= std_size` for a pooled page, `+= layout.total_size` otherwise | the pool item is `[std_size]u8`; this keeps upstream's `page_size` drift (findings 14.25): creation adds the pool buffer, `destroyNodeExt` subtracts `memory.len` = `pageListMemoryLen` = `layout(capacity).total_size` |
| `destroyNodeExt`: `v.* -= page.memory.len`; memset + pool destroy / free | asserted subtraction (Zig traps; Go-only test `TestPl5_panic_destroyNodeExt_total_size_underflow`); the rest is the garbage collector's | |
| `grow` prune: `@memset(buf, 0); first.data = .initBuf(.init(buf), Page.layout(cap))` | `clear` of the page's rows and cells, then `pageInitBuf` on them, resliced to the new capacity (new slices only if the new capacity needs more rows or cells than the old slices hold) | the reused page keeps its rows/cells as Zig keeps its buffer |
| `eraseRows` full-list case `page.reinit()` | `Page.Reinit` (page.go), which reuses rows and cells | |
| `std_size`, `PagePool.item_size` | `pageListStdSize()` / `pageListPagePoolItemSize()` = `PageLayoutOf(PageStdCapacity).total_size`, computed per call (tests change PageStdCapacity) | functions read it once into a local where they need it more than once |
| `MemoryPool.init/deinit/reset`, `pool.pages.reset(...)`, BufNode zeroing walk in `reset` | stubs / comments | `reset` still computes `page_count` (its divCeil `catch unreachable`) |
| `pool.pins.create()`, `pool.nodes.create()` | `new(PageListPin)`, `new(PageListNode)` | |
| `fastmem.rotateOnce(Row, s)` | `pageListRotateOnce` (not a Zig declaration): `tmp := s[0]; copy(s, s[1:]); s[last] = tmp`; an empty slice panics on `s[0]` as Zig traps | |
| `PinSet` (`AutoArrayHashMapUnmanaged(*Pin, void)`) | slice in insertion order; `PutNoClobber` asserts absence (std's `assert(!found_existing)`), `SwapRemove` moves the last key into the hole | linear search (see optimisations) |
| `Clone.TrackedPinsRemap.putNoClobber` | map lookup assert, then store | |

#### Pointers into pages

`ReflowCursor.page_row`/`page_cell` are Go pointers into the page's `rows`
and `cells`. Zig's many-item pointer steps (`cell + 1` in cursorForward,
`rows + 1` in cursorScroll, `rows ± n` in cursorAbsolute, `cells[rev_i]` from
`page_cell` in countTrailingEmptyCells) index the slice at
`Page.cellIndex`/`Page.rowIndex` plus the offset. `row.cells.ptr(memory)` is
`Page.rowCells(row)`. Slices of many-item pointers (`rows.ptr(..)[a..b]`) are
Go slices of `page.rows`, whose length is the row capacity, so they are
bounds-checked against the capacity where Zig is unchecked (never exceeded by
the library's own calls).

#### Control flow

- `errdefer` that changes state (`min_max_size` in resize and
  resizeWithoutReflow, `cols` and the three page-undo errdefers in
  resizeWithoutReflowGrowCols, the orphan-page cleanup in resizeCols,
  `destroyNode(target)` in split) is a deferred closure on the named error
  result, registered at the same point, so Go's LIFO order equals Zig's.
  `errdefer` that only frees memory is a comment. In Go no allocation fails,
  so of these only split's (clone failure, findings 17.29/30) is reachable.
- A block-scoped `defer` (the per-iteration `defer node.data.assertIntegrity()`
  in clone, resizeWithoutReflow, resizeWithoutReflowGrowCols, eraseRows; the
  `defer assert(total_rows == old_total_rows)` of resizeWithoutReflow's cols
  block; `pauseIntegrityChecks` around `list.increaseCapacity` in
  ReflowCursor.increaseCapacity) is a closure called in place, so the defer
  runs where Zig's does, also on an error.
- Labeled blocks: `break :x` that ends a block followed by nothing is a
  `return` from a closure (scroll's `row:` and `delta_row:`, grow's `prune:`
  returns `(node, pruned)`, resizeWithoutReflowGrowCols' `prev:`); `break
  :gt` is Go's `break` out of the switch; `fast:` is a flag; `while ... else`
  is a test of the loop variable after the loop (`node_it == nil`).
- `std.debug.runtime_safety` blocks are on (safe build), as in
  ref_counted_set.go: resize's `assert(v > 0)`, and every `if (comptime
  std.debug.runtime_safety) unreachable` in writeCell is `panic("unreachable")`;
  the unsafe-build fallback after it is dropped with a comment, so no `break
  :hyperlink` / `break :style` remains.
- `log.err` is `warn("page_list", ...)` marked `// Zig: log.err`; `log.info`
  and `log.debug` are comments (not printed), as in osc_parsers_color.go.
- `increaseCapacity`'s `inline else` over the four Capacity fields is
  written out per field; the shared body is `pageListIncreaseCapacityDouble`
  (generic over u16/u32): doubles, or saturates to maxInt when the doubling
  overflows and `old < maxInt`, or reports OutOfSpace.
- `Pin.downOverflow`/`upOverflow`'s `std.math.cast(CellCountInt, v) orelse
  maxInt` is `pageListCastCellCount`; the `.overflow` value is built by
  `pageListPinOverflowOverflow`. `Pin.left`'s cast `orelse result.x` and
  `Pin.right`'s `+|=` with `orelse maxInt` are written in place
  (`saturatingAddU16`).
- `@min(u16, usize)` in PageIterator's `.count` branches has type u16 in Zig;
  Go converts the result to `SizeCellCountInt` (`len > limit.*` therefore never
  holds, findings 14.40).
- `pointFromPin`'s `inline else` + `@unionInit` is a switch setting the
  payload field named by the tag.
- `split`: Go passes `p` by value, so upstream's aliasing hazard (findings
  14.27) cannot occur in Go; the test that documents it already builds a
  fresh pin.

#### Integer traps made explicit

Zig traps on u16/usize overflow and underflow; Go wraps. Asserts were added
wherever an operand is not bounded by an earlier check, each marked `// Zig:
... traps`: verifyIntegrity's offset subtractions, `pauseIntegrityChecks`
(Go-only test), `self.rows - c.y - 1` and the u32 `current` in resizeCols,
the `-|`/`-=`/`+=` on the cached viewport offset in scroll/eraseRowBounded/
grow (Go-only test `TestPl5_panic_eraseRowBounded_cached_offset_underflow`),
`scroll(.delta_row = minInt)` and `scrollPrompt(minInt)` negations (Go-only
tests), `size.rows - 1` / `size.cols - 1` of an empty page (getBottomRight,
nextUp, cursor helpers), `row.y - 1` in nextUp's count branch (Go-only
test), `rows - (y + 1)` in downOverflow (Go-only test), `extra % cols - 1`
in rightWrap (Go-only test), `y += rows` in `Cell.screenPoint` (u16),
`total_rows -= ...` in eraseRows/grow/resizeWithoutReflow. `.?` unwraps are
`assert(x.Valid)` / `assert(p != nil)` where the value is not dereferenced at
once.

#### Candidate B (resize)

`PageList.term51_resizing` (a field upstream does not have) implements the
deferral docs/mapping.md requires: `Resize` sets it (after registering its
`defer AssertIntegrity`) and clears it in a later defer, so the clear runs
first and the full check, the "viewport pin has enough rows" check included,
runs when `Resize` returns; `verifyIntegrity` skips only
`ViewportPinInsufficientRows` while it is set. This is
`patches-candidates/B-0001-tolerate-transient-pin-check-in-resize.patch`.

#### Tests

- 768 Zig tests in PageList.zig: 736 translated in PageList_1..6_test.go, 1 in
  rollback_test.go, 31 already in scripts/not-translated.txt (memory pool,
  allocator failure, tripwire, Kitty placeholder, cval). No layout-only test
  needed adding.
- One translated test is wrong (see the worktree's final report):
  `TestPl2_PageList_compact_oversized_page` re-evaluates Zig's
  `for (0..capacity.rows - size.rows)` bound on every iteration.
