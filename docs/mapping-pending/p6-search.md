# Mapping notes pending merge: the search module (worktree p6-search)

Sections ready to fold into `docs/mapping.md` (a new "Search module:
implementation notes" part after the Screen and Terminal module), plus the
changes other docs need.

## Search module: implementation notes

Files: `datastruct/circ_buf.zig` -> `terminal/datastruct_circ_buf.go`,
`search/sliding_window.zig` -> `search_sliding_window.go`,
`search/pagelist.zig` -> `search_pagelist.go`, `search/active.zig` ->
`search_active.go`, `search/viewport.zig` -> `search_viewport.go`,
`search/screen.zig` -> `search_screen.go`, `search.zig` -> `search.go`.
`search/Thread.zig` is not ported (`search.Thread` is `void` in the library
build; see the header of search.go). The wireframe headers (names, Go-memory
rule, widths, omissions) are followed as written; no signature changed except
`SeekBy` (below). Idioms are PageList.go's: a state-changing `errdefer` is a
deferred closure on the named error result, a memory-only `errdefer` is a
comment, `.?` is `assert(x != nil)` / `assert(x.Valid)`, `log.info` is a
comment (not printed).

### New std ports

| Zig 0.15.2 std | Go | Tests |
|---|---|---|
| `std.mem.reverse` (scalar loop; the `use_vectors` block is not ported, it swaps the same elements) | `stdMemReverse[T]` (`std_mem.go`) | `test reverse` in `std_mem_test.go` (the `u0` block uses `struct{}`, the union block a tagged struct) |
| `std.mem.rotate` | `stdMemRotate[T]` (`std_mem.go`) | `test rotate` |
| `std.ascii.isUpper`, `toLower`, `eqlIgnoreCase`, `indexOfIgnoreCase`, `indexOfIgnoreCasePos`, `indexOfIgnoreCasePosLinear`, `boyerMooreHorspoolPreprocessIgnoreCase` | `stdASCII...` (new file `std_ascii.go`) | `test eqlIgnoreCase`, `test indexOfIgnoreCase`, and the isUpper/toLower lines of `test "ASCII character classes"` in `std_ascii_test.go` |

Why ported: `bytes.EqualFold`/`strings.EqualFold` fold Unicode, and the
match index decides every search result; std's Boyer-Moore-Horspool path
(haystack >= 52 bytes and needle > 4 bytes) is ported as well, so the code
path is upstream's. `slices.Reverse` would do for `reverse`, but `rotate`
is three reversals and has no Go counterpart, so both are ported side by
side.

### `circ_buf.zig` (`datastruct_circ_buf.go`)

- `CircBuf(T, default)`: the wireframe's generic type with the `default_`
  field. `@memset(..., default)` is a loop; `alloc.realloc` in `resize` is
  a new slice of `size` elements holding the old contents (truncated when
  shrinking, as realloc does); `std.mem.rotate` is `stdMemRotate`;
  `fastmem.copy` is Go's `copy`. `deinit` sets the buffer to the Go zero
  value (`self.* = undefined`).
- `quirks.inlineAssert` is `assert`.
- `seekBy(amount: isize)`: `SeekBy(amount int64)` (project lead's ruling,
  the 64-bit reference's isize on every platform, as the scroll deltas).
  `+|= @intCast(amount)` converts with `pageListUsizeFromIsize` (saturates
  at the largest `uint` of a 32-bit target, where `+|=` would saturate
  anyway) and adds with saturation; `-|= @intCast(@abs(amount))` takes the
  u64 magnitude (`uint64(0) - uint64(amount)`, 2^63 for minInt) and
  subtracts with saturation at 0. `idx` stays `uint`.
- Integer traps made explicit: `len() + amount` (ensureUnusedCapacity),
  `offset + slice_len` (getPtrSlice, before upstream's assert),
  `storage.len - len()` in advance and in len() (a length beyond the
  capacity after a shrink). Division by a zero capacity panics in Go as it
  traps in Zig (Go-only test `TestCb2_panic_iterator_next_zero_capacity_with_len`).

### `sliding_window.zig` (`search_sliding_window.go`)

- `std.MultiArrayList(Chunk)` `chunk_buf` is a `[]HighlightFlattenedChunk`;
  `clearRetainingCapacity` is `[:0]`, `appendAssumeCapacity` is `append`
  into the reserved capacity (so a returned highlight shares the window's
  chunk memory, as in Zig), `ensureTotalCapacity(n)` makes a new slice of
  capacity exactly `n` when the capacity is smaller (std may reserve more;
  not observable).
- `highlight`'s reverse fixup: `std.mem.reverse` on `items(.node)`,
  `items(.start)`, `items(.end)` reverses three columns and leaves the
  serial column alone (findings 19.4: serials stay in data order). Go
  reverses the three fields one by one (`searchSlidingWindowReverseChunkField`,
  not a Zig declaration).
- `@intCast` of a cell map y (u32) to a chunk row (u16) is
  `searchSlidingWindowCellCount` (asserts it fits).
- `append`: `.{ .emit = .plain, .unwrap = true }` is `FormatterOptionsPlain`
  with `unwrap = true`; `formatter.point_map = .{ .map = &meta.cell_map }`
  is `Some(FormatterPageFormatterPointMap{map_: &meta.cell_map})`; the
  writer is `oscWriterAllocating`; a formatter or writer error returns
  ErrOutOfMemory as Zig's `catch` does.
- `std.debug.runtime_safety` blocks (highlight's asserts, the prune
  check, `assertIntegrity`) run unconditionally; the Valgrind check has no
  Go counterpart.
- Control flow: `slices:`, `prefix:`, `suffix:`, `formatter:` are plain
  blocks; `overlap:` and `prune:` are closures (`return` = `break :x`);
  `tl:` is a closure returning the local struct; `while ... else` is a loop
  plus a `found` flag; `else unreachable` is `panic("unreachable")`.
- Integer traps made explicit: `data_len - data_offset`, `start + len - 1`,
  `ends[0] - 1` and `starts[last] + 1` (u16) in the reverse fixup,
  `rows - 1` in append.
- Rulings: (1) empty needle: `Next` returns null at once (Go-only line at
  the top of `next`; upstream reaches `highlight(0, 0)` and crashes);
  `init` and `append` accept it. (2) the end of `next` is `len + 1 -
  needle.len` (upstream `len - needle.len + 1`, sliding_window.zig:279, which
  underflows before the `+ 1` in safe builds; the corpus records the
  release value), with an assert for the remaining unreachable case.

### `pagelist.zig`, `active.zig`, `viewport.zig`

- `PageListSearch.init`: `errdefer list.untrackPin(pin)` is a deferred
  closure on the named error; `start.data.size.rows - 1` / `cols - 1`
  asserted (u16). `feed`'s `rem -|= ...` is a saturating subtraction.
- `ActiveSearch.update` and `ViewportSearch.update`: `needle.len - 1` in the
  overlap loops (active.zig:94, viewport.zig:165 and the trailing loop) is a
  plain uint subtraction that WRAPS for an empty needle: the project lead's
  ruling follows upstream's release build, which the corpus records
  (`empty_{active,viewport}_update`, case `pages3_wrap`, `unchecked_only
  empty_needle`). It wraps to maxInt(uint), which no page's byte count
  reaches on either width, so 32-bit ARM gives the same result.
- `ViewportSearch.update`: `if (old.eql(fingerprint)) match: { ... }` is a
  closure whose `false` is `break :match`; `active:` is a closure returning
  the bool; the `errdefer { fingerprint.deinit(); self.fingerprint = null; }`
  is a deferred closure on the named error; `trailing:` is a closure that
  sets the error. `Fingerprint.toOwnedSlice` is the slice itself.

### `screen.zig` (`search_screen.go`)

- Tripwire (`reloadActive_tw`, both `tw.check` calls) and `allocator()` are
  omitted (wireframe header).
- `init`: `errdefer result.deinit()` is a deferred closure on the named
  error. `feed`'s `self.* = new` is `*self = new` after `self.Deinit()`.
- `reloadActive`: the `defer if (select_prev) ...` and the later
  `errdefer if (old_selection_idx ...)` are deferred closures registered in
  Zig's order (LIFO matches); every `self.select(...) catch |err|
  log.info(...)` ignores the error. The labeled blocks `select_prev:`,
  `history:` (a closure returning an error), `state:`, `selected:`,
  `active_prune:`, `fixup:` are closures or if-blocks. The `else => { const
  old_state = ...; defer self.state = old_state; try self.tickActive(); }`
  arm is a closure with a defer, so the state is restored also on an error.
  `replaceRangeAssumeCapacity(0, i, &.{})` is `copy` down plus a reslice;
  `shrinkAndFree(alloc, i)` is `[:i]`; `clearRetainingCapacity` is `[:0]`
  after `Deinit` of each element.
- `std.debug.runtime_safety` assert in reloadActive runs unconditionally.
- Integer traps made explicit: `prev.idx + 1` (selectNext; Go-only test
  `TestSs2_panic_selectNext_index_overflow`), `m.idx += added_len`,
  `m.idx -= old_active_len`, `m.idx += active_results.len`.
- Ruling, feed on a completed search: `Feed` returns ErrSearchComplete and
  changes nothing when the state is `.complete` and the history searcher
  would still feed data. To decide without side effects it calls
  `searchScreenHistoryWouldFeed` (not a Zig declaration): PageListSearch.feed
  itself, run on a fresh reverse window and an untracked copy of the pin
  (feed's result depends only on the pin and on `append`'s byte counts,
  which depend only on the node). When it says false, the real
  `history.feed()` runs as upstream (a complete search prunes stale history;
  with an empty needle feed appends one page and returns false, finding 16).
  The `.complete => unreachable` arm is kept and cannot be reached.
- Ruling, empty needle: nothing in screen.zig changes; the window's `next`
  returns null, so the result lists stay empty.

### Tests

- Test edits (SeekBy is int64, project lead's ruling):
  `datastruct_circ_buf_1_test.go` (translation note; test "term51:
  CircBuf.Iterator.seekBy: clamps at zero and saturates at the maximum":
  `math.MinInt`/`MaxInt` -> `math.MinInt64`/`MaxInt64`, expected values
  written as the 64-bit reference's values saturated to `uint`);
  `datastruct_circ_buf_2_test.go` (`SeekBy(int(o))` -> `int64(o)`);
  `search_sliding_window_1_test.go`, `search_sliding_window_2_test.go`
  (`SeekBy(int(i))` -> `int64(i)`).
- Two Go-only panic tests contradict the feed-on-complete ruling and fail:
  `TestSs1_panic_feed_complete_state_with_more_history_to_feed`
  (search_screen_1_test.go) and
  `TestSs2_panic_Feed_complete_state_with_history_to_feed`
  (search_screen_2_test.go) expect the `.complete => unreachable` panic; the
  ruling (and `TestSearchRuling_feed_complete_*`) wants ErrSearchComplete.
  They should be removed (or turned into ErrSearchComplete checks) and
  their rows in docs/go-only-tests.md dropped.
- New test files for std: `std_mem_test.go` (2 tests), `std_ascii_test.go`
  (3 tests); not checked by scripts/check-test-mapping.py (std files).

## For docs/post-port-optimizations.md

- `searchScreenHistoryWouldFeed` formats the pages a complete-state feed
  would add a second time (only in the `.complete` state).
- `searchSlidingWindowReverseChunkField` copies each column to a temporary
  slice; three in-place swap loops would do.
- `SlidingWindow.append` encodes into a temporary writer and copies into
  the circular buffer (upstream's own note); likewise the point map.
- `stdASCIIIndexOfIgnoreCase` is called on each of the two data slices and
  the overlap buffer on every `next`; upstream's own design.
