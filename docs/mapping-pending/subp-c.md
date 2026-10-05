# Mapping notes from impl/subp-c (to merge into docs/mapping.md)

Files: `osc_parsers_context_signal.go`, `osc_parsers_kitty_clipboard_protocol.go`,
`osc_parsers_kitty_text_sizing.go`, `osc_parsers_iterm2.go`
(`src/terminal/osc/parsers/*.zig`).

## Language features (additions to the table)

| Zig | Go | Notes |
|---|---|---|
| `std.meta.stringToEnum(E, s)` | loop over the enum's values comparing `s` with `String()` | exact, case-sensitive match on the tag name, as Zig |
| `std.mem.splitScalar(u8, s, c)` iterated to the end (or until a `return`) | `for _, piece := range bytes.Split(s, []uint8{c})` | same pieces, empty ones included; one empty piece for empty input. `it.next() orelse` after a fixed number of `next()` calls becomes a length check on the `bytes.Split` result. The pieces' capacity is clipped by `bytes.Split`; no caller reads past a piece |
| `std.mem.indexOfScalar` / `indexOfScalarPos(s, pos, c)` | `bytes.IndexByte(s)` / `pos + bytes.IndexByte(s[pos:], c)` | `-1` is `null` |
| `std.mem.startsWith`, `std.mem.eql` | `bytes.HasPrefix`, `bytes.Equal` | |
| `std.mem.trim(u8, s, &std.ascii.whitespace)` | `bytes.Trim(s, " \t\n\r\x0b\x0c")` | the cutset is all ASCII, so `bytes.Trim` works byte by byte. It returns nil instead of an empty slice when everything is trimmed; the port only looks at the length and the bytes, and wraps it in `Some(...)` where Zig gives a non-null empty slice |
| `std.ascii.isWhitespace` | file-private helper (`' '`, `'\t'`...`'\r'`) | |
| `std.mem.indexOfNone(u8, s, set) == null` | loop: every byte is in `set` | |
| `std.StaticStringMapWithEql(V, std.ascii.eqlIgnoreCase)` (`iterm2.Map`) | Go map keyed by the ASCII-lowercased key; `get` lowercases the lookup key the same way (only `A`-`Z` change; non-ASCII bytes are kept, unlike `strings.ToLower`) | gives the same result as Zig's length-indexed `eqlIgnoreCase` scan because no two keys differ only in case; the map's start-up builder asserts this |
| `std.fmt.parseInt(u64, s, 10)` after an all-digits check (`context_signal`) | `strconv.ParseUint(s, 10, 64)` | with only digits left, both fail exactly on "" and on overflow |
| `std.fmt.parseInt(u4, s, 10)` (`kitty_text_sizing.OSC.update`) | file-private port of Zig 0.15.2 `parseIntWithGenericCharacter`/`parseIntWithSign`, returning `(value, ok)` | strconv is NOT a substitute: Zig accepts a leading `+`, `-0` (`-000` too), `_` between digits (`1_0` = 10) and accumulates in a u8 with overflow checks before the u4 cast. The caller maps every error to `error.InvalidValue`, so the helper returns `ok == false` instead of `InvalidCharacter`/`Overflow` (no new sentinels needed) |
| `std.math.cast(u3, v) orelse ...`, `std.enums.fromInt(E, v) orelse ...` | explicit range check (`v > 7`, `v > <last value>`) | |
| Comptime-typed result `option.Type()` / `key.Type()` | the wireframe's `...FieldValue` / `...OptionValue` struct; `Read` fills only the field named after the requested key | a nested `switch` picks the field |
| Labeled block with `break :value x` / `continue` / `return` inside | a flag or straight-line `if ... continue` | control flow unchanged |
| `[:0]` sentinel slices (`data[a..b :0]`) | plain sub-slices `data[a:b]` that keep their capacity, so the NUL written just before stays at `s[:len+1][len]` | the sentinel check Zig does when slicing is not repeated: in all three places the NUL was written by the same function on the line before, so the check can't fail |
| `log.debug(...)` (iterm2) | not printed (comment left) | like `log.info` in `search_screen.go` |
| `log.warn("... {}", .{err})` | `warn(..., "error.%v", err)` | Zig prints an error value as `error.Name` |
| `quirks.inlineAssert` | `assert` | |

## Per-file notes

### `osc/parsers/context_signal.zig`
- `Field.read` returns `OSCParsersContextSignalFieldValue`; the numeric arm
  (a labeled block in Zig) is inlined in `Read`, with no helper function.
- The empty-metadata literal `""` is `[]uint8("")` (non-nil, length 0).

### `osc/parsers/kitty_clipboard_protocol.zig`
- `Option.read`'s `value:` block is a loop that either returns the null
  result or sets `value` and a `found` flag. The final `switch` fills the
  field named after `key`.
- `parse` keeps upstream's `assert(parser.state == .@"5522")`.

### `osc/parsers/kitty_text_sizing.zig`
- `.{ .text = payload }` sets `scale: 1` explicitly (Zig's default).
- `update` parses the value before looking at the key, as upstream does, so an
  unknown key with a bad value reports `InvalidValue`.
- The two `it.next()` calls on the `'='` iterator are `it[0]` (always present:
  the "missing key" branch is kept but can't run, as upstream's can't)
  and `it[1]` (missing when the piece has no `'='`). Anything after a second
  `'='` is ignored, as upstream does.

### `osc/parsers/iterm2.zig`
- `map` (Go `oscParsersIterm2Map_`) is built at package init by
  `oscParsersIterm2MapInitComptime`, the counterpart of `Map.initComptime` plus
  the comptime block that lists the fields of `Key`.
- Unknown and unimplemented keys set `parser.command = .invalid` and return
  null **without** setting `parser.state = .invalid` (upstream quirk, kept).
- The trailing `return &parser.command;` after the exhaustive `switch` is
  unreachable upstream and is kept.

## New std stand-ins that could move to zig.go later
These are file-private now so they can't clash with other worktrees:
`oscParsersKittyTextSizingParseIntU4` (Zig `parseInt` for u4),
`oscParsersKittyClipboardProtocolIsWhitespace`,
`oscParsersKittyClipboardProtocolAsciiWhitespace`, `oscParsersIterm2AsciiLower`.
If other sub-parsers (osc9 etc.) also port `std.fmt.parseInt`, a shared
generic `zigParseInt` in `zig.go` could replace them after the merge.
