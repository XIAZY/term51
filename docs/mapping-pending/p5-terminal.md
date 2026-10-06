# Mapping notes: Terminal (worktree p5-terminal)

Sections ready to merge into docs/mapping.md (Screen and Terminal module).

### `Terminal.zig` (`Terminal.go`)

File: `terminal/Terminal.go`. The wireframe header (Go-memory rule, widths,
packed structs, omitted kitty-graphics code, Zig defaults) is followed as
written; no signature changed. The notes below are what the implementation
adds; the file header's "Implementation notes" section says the same in
short.

#### Control flow

| Zig | Go | Notes |
|---|---|---|
| `defer self.screens.active.assertIntegrity()` (print, printCell), `defer self.screens.active.pages.untrackPin(cur_p)` (insertLines, deleteLines) | `defer func() { self.screens.active.F() }()` | Go evaluates a deferred call's receiver when the `defer` statement runs; Zig evaluates the whole expression at scope exit. |
| `defer` inside an `if` block (`pauseIntegrityChecks(true); defer pauseIntegrityChecks(false)` in insertBlanks and deleteChars) | a closure called in place | the PageList idiom: the defer runs at the end of the block, as in Zig |
| `defer if (...) {...} else {...}` in index | deferred closure | runs also on the error returns, as Zig's `defer` |
| `errdefer self.screens.active.cursor.style = old_style` (decaln) | deferred closure on the named error result `err` | the PageList idiom |
| `errdefer screen_set.deinit(alloc)` (init) | comment | only frees memory |
| labeled block `grapheme:` in print, left by `break :grapheme` from inside the block | `goto grapheme`, label after the `if` block | mapping's goto rule; no variable comes into scope |
| labeled block `transfer_graphemes:` (`orelse break :transfer_graphemes`) | `goto transfer_graphemes`, label after the inner block | same |
| `wide:` / `narrow:` blocks that are a whole switch prong (print, printCell) | Go `break` out of the switch | as in osc_parsers_semantic_prompt.go |
| `click:` (semanticPrompt), `at_prompt:` (eraseDisplay): blocks followed by nothing | closure called in place, `break :x` is `return` | the PageList idiom; the `while (it.next()) ... else break :at_prompt` is a labeled `for` whose null branch returns, and `break` out of the loop is `break loop` |
| other labeled blocks producing a value (`prev:`, `left:`, `brk:`, `c:`, `end:`, `cells:`, `wrap_mode:`, `new:`) | a Go block assigning a variable declared just before | label kept as a comment |
| local types `Prev`, `WrapMode`, the `desired_wide` enum, the `params` struct | local Go `type Prev struct`, local `const` blocks (iota), a local anonymous `struct` variable | |
| `self.gpa()` | `gpa()` returns nothing (wireframe); VtStream calls it and passes `terminalGpa{}` (an added empty type, NOT a Zig declaration) to `StreamInitAlloc`, which only tells nil from non-nil | the stream holds `&handler`, a pointer to its own copy of the value Zig stores |
| `catch \|err\| switch (err) { ... }` over `Page.CloneFromError` (insertLines and deleteLines, inline twice) | `terminalCloneFromErrorAdjustment(err)` (NOT a Zig declaration), one helper for both | `default: panic("unreachable")` for an error outside the set; the `catch \|e\| switch (e)` on increaseCapacity is written in place (`ErrOutOfMemory`, `ErrOutOfSpace`, else unreachable) |
| `std.unicode.Utf8View.init(str)` + iterator (printString) | `utf8.Valid`, then `range string(str)` | the osc_encoding.go equivalence; invalid input returns `ErrInvalidUtf8` before anything is printed, as Zig |
| `std.io.fixedBufferStream(buf)` writer (printAttributes) | a local `writeAll` closure over `buf` and `pos`: copies what fits, `ErrNoSpaceLeft` if not all fits | the bytes left in `buf` on failure equal Zig's (std's writeAll writes until the buffer is full, then fails). `print` formats with `itoa` first, then writes. `input_key_mods.go`'s `inputKeyModsFixedBufferStream` is still a wireframe of another file, so it is not reused (could be shared later) |
| `log.warn` / `log.err` | `warn("terminal", ...)`, `// Zig: log.err` where it is `err`; `std.log.warn` (index) is `warn("default", ...)`; `{}` of an error is `error.%v` | `log.debug` / `log.info` are comments |

#### Pointers

`[*]Cell` many-item pointers that Zig steps (`left - 1`, `left + n`, `x -= 1`
in insertBlanks; `x += 1` in deleteChars; `cells - cursor.x` in eraseLine;
`cells[0..count]` in eraseChars; `spacer_cell[0..1]` / `wide_cell[0..1]` in
printCell) are indexes into the cursor page's `cells`, from
`Page.cellIndex(cursor.page_cell)`. They are bounds-checked against the whole
page (Zig is unchecked; never exceeded by in-range state). insertBlanks'
backwards walk uses a signed `int` index, because Zig's pointer steps to one
cell before `left` to end the loop, which for the first cell of a page would
wrap a Go `uint`. `row.cells.ptr(page.memory)` (rowWillBeShifted, decaln) is
`Page.rowCells(row)`, which reaches to the end of the page as Zig's pointer
does: rowWillBeShifted's `cells[right + 1]` for a wide cell in the last column
touches the next row's first cell in both.

#### Integer widths and traps made explicit

- `@min(a, b)` of a `usize` and a `CellCountInt` has type `CellCountInt` in
  Zig (it narrows to the smaller bound); Go converts the result:
  `adjusted_count` (insertLines, deleteLines, insertBlanks), `count`
  (deleteChars), `end` (eraseChars), the `x`/`y` of setCursorPos, `bottom` /
  `right` (setTopAndBottomMargin, setLeftAndRightMargin), and the argument of
  the screen's cursorLeft in cursorLeft.
- Asserted where Zig traps and Go would wrap: `col + x_offset` and
  `row + y_offset` (usize) in setCursorPos (Go-only tests
  `TestTm5_panic_setCursorPos_*_overflow_in_origin_mode`);
  `scrolling_region.left - 1` in rowWillBeShifted (Go-only tests
  `..._rowWillBeShifted_spacer_tail_at_left_*zero`); `prev.left - 1` in
  print's narrow branch; `right_limit - scrolling_region.left` in print's
  wide case; `rows - 1` / `cols - 1` in init and resize (after the calls
  that Zig makes first, so a lower layer may panic first, as in Zig).
- `.?` unwraps are `assert(x.Valid)` / `assert(p != nil)`: the grapheme
  lookups in print, `page_pin.down(rem - 1).?` (insertLines),
  `cur_p.up/down(adjusted_count).?`, `screens.get(.primary).?` (resize,
  switchScreen), `getCell(pt).?` (isDirty).
- `assert(!unicode.graphemeBreak(cp1, cp2, &state))` keeps its side effect
  on `state` (Go's `assert` always evaluates its argument, as Zig's safe
  build does).
- `@intFromBool` is written as an `if`.
- `-| 1` (setCursorPos) is `if x > 0 { x -= 1 }`.

#### Zig defaults written out

- `init`: `ScreenSet.init(.{ .cols, .rows, .max_scrollback })` also gets
  `kitty_image_storage_limit = ScreenOptionsDefault.kitty_image_storage_limit`
  (Zig's field default); `modes = .{ .values, .default }` keeps
  `saved = ModesModePackedInit()`; `mouse_shape = MouseShapeText`,
  `flags = TerminalFlags{focused: true}`.
- `resize` of the alternate screen (`.reflow = false`) writes
  `prompt_redraw: OSCParsersSemanticPromptRedrawFalse`, the field default.
- `restoreCursor`'s fallback `.charset = .{}` is
  `ScreenCharsetState{gr: CharsetsSlotsG2}`; `fullReset`'s `flags = .{}` is
  `TerminalFlags{focused: true}`.
- `TerminalColorsDefault` is initialised at start-up from
  `ColorDynamicRGBUnset` and `ColorDynamicPaletteDefault`.
- printCell's `if (comptime std.debug.runtime_safety) cell.wide = .narrow`
  is unconditional (safe build).

#### Tests

Terminal_1..10_test.go: 1,213 Go tests, each run on its own (a
not-implemented panic aborts a whole-file run), natively (arm64) and under
32-bit ARM qemu with identical results: 867 pass (all 30 Go-only panic tests
among them), 0 fail, 335 blocked on formatter.go (`ScreenFormatterInit`,
reached through `plainString` / `Screen.dumpString*`), 11 blocked on
stream_readonly.go (`StreamReadonlyHandlerInit`, through `vtHandler` /
`vtStream`).

One translated test re-evaluates a loop bound Zig evaluates once:
`TestTm4_print_clustering_VS16_transfers_grapheme_data_across_a_page_boundary`
(Terminal_4_test.go:1768, `i < uint(term.screens.active.pages.pages.first.data.capacity.rows)`
in the loop condition) for Zig's
`for (rows..t.screens.active.pages.pages.first.?.data.capacity.rows)`
(patched Terminal.zig:15079). It passes, because the first page's capacity
does not change while `index` runs; the Devanagari test of the same shape
(Terminal_1_test.go:1567) hoists the bound correctly.
