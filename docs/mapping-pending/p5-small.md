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
