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
| 7 | `std_hash_auto_hash.go` `.Deep` slice hash | Calls Wyhash `update` once per byte, then the length. One `update` of the bytes plus the 8 length bytes gives the same hash. | Literal translation of `autoHashStrat` over a slice (one `hash` call per element). |
| 8 | `std_hash_wyhash.go` `final` | Copies the state (`shallowCopy`) even when `final` is the last call. | Zig's `final` takes `*const Wyhash` and copies; kept literal. |
| 9 | `RefCountedSet.Insert` | Does the runtime-safety "not already present" lookup before adding, so every new item costs two lookups. | Upstream's `if (comptime std.debug.runtime_safety)` assert is on in the reference build. |
| 10 | `RefCountedSet` with `HyperlinkSetContext` | `any(ctx).(RefCountedSetContextDeleted[T])` on each delete may allocate (the context holds two pointers). A per-instantiation function value set in `RefCountedSetInit` would avoid it. | Go cannot call a method outside the type constraint; `has_deleted` is decided once at init (compile-time rule), the call still converts. |
| 11 | `RefCountedSet.capacityForCount` | Float division and `ceil`; `((n+1)*16 + 12) / 13` is the same for every recorded capacity. | Zig's float formula, kept literal. |
| 12 | `RefCountedSetInit` | Fills `items` with a loop. A template copy would be cheaper. | Literal translation of the init loop. |
| 13 | `style.go`/`hyperlink.go` VT and HTML `Format` | Build an allocating writer per call, and `formatColor` builds a string before `writeAll`; `formatter.go` could write into the caller's writer. | Each Zig `writer.print` is one `writeAll` of the same bytes, so partial output on a full fixed writer is identical. |
