# Mapping notes from impl/subp-a (OSC sub-parsers)

Files: `osc_parsers_color.go`, `osc_parsers_kitty_color.go`,
`osc_parsers_mouse_shape.go`, `osc_parsers_change_window_icon.go`,
`osc_parsers_change_window_title.go`, `osc_parsers_clipboard_operation.go`,
`osc_parsers_report_pwd.go`, `osc_parsers_rxvt_extension.go`,
`osc_parsers_hyperlink.go`.

## Per-file notes: `osc/parsers/*.zig` (this group)

- `parser.writer orelse { ... }` is `writer := parser.writer; if writer == nil`.
  `writer.writeByte(0) catch` is `if err := writer.writeByte(0); err != nil`
  (`writeByte` and `buffered` are osc.go's `oscWriter` methods).
- `parser.alloc orelse` (color, kitty_color) is `if parser.alloc == nil`. The
  allocator value itself is never used: the allocator parameters of
  `parseColor`, `List.append` and so on are dropped.
- NUL-terminated slices `data[a .. b :0]` are the Go sub-slices `data[a:b]`.
  They point into the parser buffer, so the NUL stays inside their capacity
  (the "byte after the slice" requirement). Zig's safety check
  `data[b] == 0` always holds at every such place in these files: the NUL
  was written just before, either by `writeByte(0)` or by `data[x] = 0`. So
  the port does not assert it.
- `assert(parser.state == ...)` (mouse_shape, clipboard_operation,
  kitty_color) is `assert(...)` from zig.go.
- `std.mem.indexOfScalar` is `bytes.IndexByte` (-1 is null). `std.mem.eql`
  is `bytes.Equal`. `std.mem.trim(u8, s, " ")` is `bytes.Trim(s, " ")`; for
  a one-byte ASCII cutset it is byte-wise, the same as Zig's, and an
  all-space input gives a length-0 result in both.
  `std.mem.indexOfScalarPos(s, pos, c)` is written out as
  `bytes.IndexByte(s[pos:], c) + pos`. Where `pos >= len(s)` is reachable
  (hyperlink), Zig's early `return null` is an explicit check.
  `std.mem.replaceScalar` is written out as a loop.
- `log.warn` is `warn("<scope>", ...)` with Zig's `{s}` as `%s`. The one
  `log.info` (color.parse, when parseColor fails) is not printed, the same
  as search_screen.go's `log.info`.

### `osc/parsers/color.zig`
- `Operation`, `Request` tag and `Target` tag: `String()` reads a name table
  indexed by value. A value outside the enum panics, as `@tagName` on an
  invalid value of an exhaustive enum does, and as `ParserState.String`
  does.
- `std.SegmentedList(Request, 2)` (`List`) is one Go slice; the segments
  cannot be observed. The wireframe declared `append`, `count`, `at`,
  `constIterator`, `deinit` and the const iterator's `next`. The port adds
  `addOne`, which is what color.zig actually calls; `append` is written as
  `addOne` plus a store, as in std. The new element of `addOne` is the Go
  zero value (Zig: undefined). `at` asserts `i < count()` as std does.
  `deinit` resets the list to its zero value (Zig: `self.* = undefined`).
  `constIterator` does not check `start_index`, matching std's `set`.
- Allocation failure: `addOne`/`append` keep Zig's `Allocator.Error` in the
  signature but always return nil. The `try` paths after them are kept,
  including the `errdefer result.deinit(alloc)` arms (written as
  `result.deinit()` before the error return). parseGetSetDynamicColor has
  no errdefer in Zig and none in Go.
- `std.mem.tokenizeScalar` / `TokenIterator(u8, .scalar)` is
  `oscParsersColorTokenizeScalar` / `oscParsersColorTokenIterator`, a
  wireframe stand-in. Zig's `next` is `peek` followed by an index advance;
  `peek` is written out inside `next` because nothing else uses it.
- `std.fmt.parseInt(u9, s, 10)` is `oscParsersColorParseIntU9`
  (parseIntWithGenericCharacter) plus `oscParsersColorParseIntWithSignU9`
  (parseIntWithSign with Accumulate = Result = u9, base 10). Both return
  `Optional[uint16]`, because every caller discards the error with a bare
  `catch`. The quirks are kept: a leading `+`, `-0` (any run of zeros after
  `-`), leading zeros, `_` between digits, and rejection of a
  leading/trailing `_` and of a bare sign. A u9 overflow (> 511) is null.
- `std.math.cast(u3, x) orelse` is `if x > 7`.
  `std.meta.intToEnum(SpecialColor, x) catch` is
  `if x > uint16(ColorSpecialItalic)`, because the values of `color.Special`
  are 0..4 with no gaps. Both checks are kept, one after the other, as in
  Zig. `std.math.cast(u8, color)` is `color <= 255`.
- `comptime op: Operation` (parseGetSetAnsiColor, parseResetAnsiColor) is a
  run-time parameter. Zig's `else => comptime unreachable` arms are
  `panic("unreachable")`. parseColor's exhaustive switch on `op` ends with
  `panic("unreachable")` for a value outside the enum.

### `osc/parsers/kitty_color.zig`
- `std.mem.splitScalar` / `SplitIterator(u8, .scalar)` is
  `oscParsersKittyColorSplitScalar` / `oscParsersKittyColorSplitIterator`,
  with std's `next` and `rest`. `index: ?usize` is `Optional[int]`.
- `list` (`std.ArrayList(Request)`, `.empty`) is the Go slice
  `KittyColorOSC.list`, starting at nil. `list.append(alloc, x) catch |err|
  { log.warn(...); continue; }` is `*list = append(*list, x)`; Go's append
  cannot fail, so the catch arm has no counterpart. The capacity that
  "{any}" prints comes from osc.go's `oscKittyColorListCapacity` (osc-core),
  not from the Go slice.
- `RGB.parse(value) catch |err| switch (err) { error.InvalidFormat => ... }`
  is a switch on the returned sentinel. Its `default` is
  `panic("unreachable")`, because Zig's error switch is exhaustive over
  `error{InvalidFormat}`.
- `list.items.len >= @as(usize, kitty_color.Kind.max) * 2` is
  `len(*list) >= int(KittyColorKindMax)*2`.
