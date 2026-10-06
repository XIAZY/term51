### `formatter.zig`

File: `terminal/formatter.go`. The wireframe header (names, `Format(writer)
error` instead of `Format() string`, Go-memory rule for the pin and point
maps, Zig defaults) is followed as written; the notes below are what the
implementation adds.

#### std.Io facilities (added to `osc.go`, Zig 0.15.2 names and code)

| Zig | Go | Notes |
|---|---|---|
| `Writer.countSplat`, `Writer.writeSplat` | `oscWriterCountSplat`, `(*oscWriter).writeSplat` | ported as written (fast path into the buffer, else `drain`) |
| `Writer.splatByte`, `Writer.splatByteAll` | `(*oscWriter).splatByte`, `splatByteAll` | `n` is a usize in Zig, `int` in Go (blank-cell counts) |
| `Writer.printUnicodeCodepoint` (what `{u}` prints) | `(*oscWriter).printUnicodeCodepoint(c uint32)` | `std.unicode.utf8Encode` with the U+FFFD fallback is `utf8.EncodeRune`, which encodes and replaces exactly the same values (surrogate halves, above U+10FFFF) |
| `std.Io.Writer.Discarding` (`init`, `drain`, field `count`) | `oscWriterDiscarding`, `oscWriterDiscardingInit`, `oscWriterDiscardingDrain`; a third vtable value `oscWriterVTableDiscarding` | Go cannot reach the Discarding from its writer in `drain` (`@fieldParentPtr`), so `count` is a field of `oscWriter` used only by this vtable: Zig's `discarding.count` is `discarding.writer.count`. `fullCount` and `sendFile` have no caller and are not ported |

#### Other std pieces

| Zig | Go | Notes |
|---|---|---|
| `writer.print(fmt, args)` | one `writer.writeAll` of the same bytes, Zig's print line in a comment above | style.go precedent. `{d}` is `itoa`, `{x:0>2}` of a u8 is `formatterHex2` (two lowercase hex digits), `{s}`/`{c}` are concatenation; `print("{f}", .{x})` is `writeAll(x.Format())`. For the fixed, allocating and discarding writers, writing the pieces one after the other gives the same bytes and the same error as one write (fixedDrain fills to the brim either way) |
| `std.io.fixedBufferStream(&buf)` + `.writer()` + `getWritten()` (formatWithState's HTML/VT header) | `oscWriterFixed(buf[:])`, `buffered()` | every write is `catch return error.WriteFailed`, and a fixed writer's failure is ErrWriteFailed; the unimplemented `inputKeyModsFixedBufferStream` (input_key_mods.go) is not used |
| `std.unicode.Utf8View.init(s) catch unreachable` + `iterator().nextCodepoint()` | `utf8.Valid(s)` (false panics "unreachable") + `utf8.DecodeRune` loop | Go's validation rejects exactly what Zig's does (overlong, surrogates, > U+10FFFF, truncated); on valid input both decode the same codepoints |
| `ArrayList.appendNTimes`; `ensureUnusedCapacity` + `appendNTimesAssumeCapacity` | `formatterAppendNTimes[T](list *[]T, v T, n uint64)` | `catch return error.WriteFailed` on list growth cannot happen in Go |
| `std.enums.values(charsets.Slots)` | package var `formatterCharsetsSlotsValues` | built once |
| `inline for (@typeInfo(modespkg.Mode).@"enum".fields)` + `@field(modes.default, field.name)` | loop over `modesModeFields`; field i is entry i, so the default is `*modesModePackedFields[i](&modes.default_)` | start-up tables of modes.go, no per-call name lookup. `@bitCast(@intFromEnum(mode))` to ModeTag is `ModesModeTagFromInt` |
| `ScreenFormatter.Extra.isSet` (backing integer != 0) | OR of the six bools | |
| Enum/tag `String()` | names arrays `formatterFormatNames`, `formatterCodepointMapReplacementTagNames`, `formatterScreenFormatterContentTagNames` | |

#### Control flow

- Labeled blocks left from inside nested statements are `goto <label>` with
  the label right after the block (page.go precedent): `palette:` in
  TerminalFormatter.format (`break :palette` on `.plain`), and `blank:`,
  `style:`, `hyperlink:` in formatWithState's cell loop. Each block is a Go
  `{ }` block so no goto jumps over a declaration.
- `cells_subset:` is written inline; its `continue` (spacer head on the
  first row) continues the row loop, as in Zig.
- `replacement:` (writeCodepointWithReplacement) is a variable set before a
  `break` out of the search loop (`// break :replacement`).
- `inline .codepoint, .codepoint_grapheme => |tag|` passes `cell.content_tag`
  as writeCell's ordinary `tag` parameter.
- `current_hyperlink_id == link_id_` (two `?u16`) is written out: same
  validity, and equal values when valid.
- The pin copies in TerminalFormatter/ScreenFormatter (`.{ .node, .x, .y }`)
  build a new PageListPin, so `garbage` is false as in Zig.

#### Zig traps made explicit

Asserts marked `// Zig: ... traps`: `terminal.rows - 1` / `cols - 1`
(scrolling region), `x + 1` on u16 (`formatterAddOneU16`: region margins,
cursor CUP), `@intCast(coord.y)` to u16 in PageListFormatter,
`@intCast(y_offset_usize)` to u16 and the u32 `start.y + y_offset` in the
blank-row point mapping (reachable in Zig: more than 65,535 accumulated
blank rows with a pin map traps), `getBottomRight(.screen).?`,
`lookupGrapheme(cell).?`, `assert(m.map.items.len > 0)` in
formatStyleClose/formatHyperlinkClose. `cells[row_start_x..row_end_x]` out of
order and `items[items.len - 1]` on an empty map (closing `</div>`) panic in
Go by themselves. Every `unreachable` prong is `panic("unreachable")`;
exhaustive switches get a `default: panic("unreachable")`.

#### Tests

- formatter_1/2/3_test.go: 197 Go tests. On this branch 2 pass and 195 are
  blocked: they build a Terminal (`TerminalInit` is a wireframe on main) and
  feed it through `t.vtStream()` (`stream_readonly.go` is still a wireframe
  on main AND on impl/p5-terminal, 9 not-implemented bodies). With
  impl/p5-terminal merged and a throw-away (uncommitted) stream_readonly
  handler, all 197 pass natively and under 32-bit ARM qemu.
- Screen_1..5, ScreenSet, ScreenSet_2, StringMap: all 633 pass natively and
  on ARM (the 187 that waited for formatter.go included).
