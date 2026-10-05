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
