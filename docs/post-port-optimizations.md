# Optimisations deferred until the port passes

The port is literal on purpose, and some of the literal translations are
slower in Go than the original is in Zig. Nothing here may be changed until
every translated test and every recorded case passes; then each item is its
own change, measured before and after.

| # | Where | What is slow | Why it is that way now |
|---|---|---|---|
| 1 | `Parser.Next` | Returns `[3]Optional[ParserAction]` per input byte. `ParserAction` is a struct holding every payload, and will carry a whole OSC command by value once `osc.zig` is ported, so three of them are copied for every byte. | Zig returns `[3]?Action`, a small tagged union; the struct-with-tag mapping makes it large. |
| 2 | `Parser.Next` | A deferred closure runs per byte to set the next state. | Zig's `defer self.state = next_state` costs nothing; Go's `defer` with a closure does. |
| 3 | Zig std stand-ins | `tokenize`/`splitScalar`/`parseInt`/whitespace helpers are duplicated per file with a file prefix (`osc_parsers_color.go`, `osc_parsers_kitty_color.go`, `osc_parsers_kitty_text_sizing.go`, `osc_parsers_kitty_clipboard_protocol.go`, `osc_parsers_iterm2.go`, `osc_parsers_osc9.go`, `osc_parsers_semantic_prompt.go`, the end of `color.go`). Not slow, but repeated: they can become shared helpers in `zig.go`. | The parser module was ported in parallel branches; each kept its helpers private to avoid name clashes. |
| 4 | `x11_color.go` lookup | Allocates a lower-cased copy of each looked-up name. | The map is keyed by the lower-cased name to give Zig's case-insensitive lookup. |
| 5 | `color.go` 256-bit set iterator | Tests every bit; could skip whole empty words. | Literal translation of Zig's `StaticBitSet` iterator use. |
| 6 | `color.go` decimal float parse | Rebuilds a string to hand to `strconv`. | Zig's float grammar is ported for validation; decimal values then go through `strconv` (both round correctly). |
| 7 | `modes.go` `ModeState` get/set/save/restore | Looks the mode up in the entries table on every call (about 420 ns per `Get` on arm64); `Terminal.print` calls `modes.get` for every printed character (Terminal.zig:326, 339, 425). | Zig resolves the mode to a struct field at compile time (`inline else` + `@field`); the port does the same lookup at run time. Build the mode-to-field mapping once at start-up. |
