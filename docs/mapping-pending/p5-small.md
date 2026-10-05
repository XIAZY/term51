# Mapping notes: cursor, StringMap, highlight, Selection, ScreenSet (worktree p5-small)

Sections ready to merge into docs/mapping.md (Screen and Terminal module).

### `cursor.zig` (`cursor.go`)
- One enum. `String()` reads the name table `cursorStyleNames` (house style
  of `ansi.go`); a value outside the enum panics on the index, as `@tagName`
  of an invalid exhaustive-enum value traps.

### `StringMap.zig` (`StringMap.go`)
- `deinit` only frees. Kept as an empty `Deinit` (the tests call it); the
  receiver is a copy, as in Zig, so there is nothing to reset. The
  oniguruma part is not ported (wireframe header).

### `highlight.zig` (`highlight.go`)
- `Flattened.init` is not ported (mapping.md already says so: the body does
  not compile upstream and nothing references it). The wireframe stub
  `HighlightFlattenedInit` is removed and replaced by a comment, as
  `PageList.diagram` was; the tests use their own `ssHlFlatten`.
- `Tracked.init`: the `errdefer screen.pages.untrackPin(start_tracked)`
  changes state, so it is a deferred closure on the named error result
  (the PageList idiom). The second errdefer is registered after the last
  failable call and so cannot run; it is a comment.
- `std.MultiArrayList(Chunk)`: `slice.items(.f)[i]` is `chunks[i].f`;
  `clone` is a fresh slice with a copy of the elements; `deinit` sets
  `chunks` to nil (Zig leaves it undefined).
- `endPin`/`untracked`: `end - 1` on the u16 chunk end is asserted
  (`assert(end > 0)`), since Zig traps on the underflow. With no chunks the
  Go index panics where Zig's does (Go-only tests). In `untracked` the
  start pin is built first so that an empty list panics on `nodes[0]`
  first, as in Zig.

### `Selection.zig` (`Selection.go`)
- `Bounds` is the wireframe's tagged struct; every exhaustive `switch
  (self.bounds)` gets `default: panic("unreachable")`. `startPtr`/`endPtr`
  on an untracked selection return the address of the pin inside the
  selection (`|*v| &v.start`), so they act on the caller's variable.
- `.?` on `pointFromPin(.screen, ...)` and `pages.pin(...)` is
  `assert(x.Valid)` (Go-only tests for order/contains/containedRow). The
  `.screen` payload access after it is not asserted: `pointFromPin` returns
  the tag it was asked for.
- `track`: `assert(!self.tracked())` guards the `bounds.untracked` access
  (Zig's tag check); the first `errdefer untrackPin` is a deferred closure
  on the named error result (PageList idiom), the second is a comment (it
  is registered after the last failable call).
- Labeled blocks `pin: { var p = ...; p.x = ...; break :pin p; }` and
  `start:`/`end:` copies in containedRowCached are locals built before the
  call; the label is kept as a comment.
- `adjust(.down)`'s `while (current.down(1)) |next| : (current = next) {
  ... } else { self.adjust(s, .end_of_line); }` is a `for` loop whose
  null branch runs the `else` and breaks; `break` on a found row skips it,
  as in Zig.
- `s.pages.cols - 1` (containedRowCached) and `size.cols - 1`
  (`.end_of_line`) are asserted `> 0` (u16 underflow traps in Zig), as
  PageList.go does for the same expressions. `@intCast(cells.len - 1)` in
  `.end` is a plain conversion: the row has a cell with text, so it is not
  empty, and its length is at most the u16 column count.
