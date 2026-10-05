# Zig to Go mapping

This port follows the Zig source of libghostty-vt (Ghostty v1.3.1, commit
332b2aef) file by file and function by function. This document is the single
record of every place where Go cannot say what Zig says, and what is used
instead. A reviewer checking a Go file against its Zig file should need
nothing else.

## Governing rules

Set by the project lead; they take precedence over anything below.

1. One Go file per Zig file, with the same name (`Parser.zig` -> `Parser.go`).
   One Go function per Zig function, in the same order, with the same control
   flow. Upstream's comments are carried over.
2. No redesign and no optimisation during the port.
3. Ordinary Go memory. Pages hold Go slices of rows and cells; references are
   Go pointers or indexes; the garbage collector frees. No raw byte blocks, no
   offset arithmetic. Functions that exist only to compute byte layouts, map
   memory or hand it back have no Go counterpart. Capacity limits are kept as
   counts so behaviour that depends on them (for example when a page splits)
   is unchanged. This is the governing rule for `page.zig` and `PageList.zig`.
4. Tests that only check memory layout are not translated; each is listed
   with its reason for the project lead's approval. Every other test is
   translated: one Go test per Zig test, same name, same assertions.
5. Where the Go standard library has a direct equivalent of a component, it
   is used behind a wrapper that keeps upstream's name, and upstream's tests
   for that component run against the wrapper. If they fail, the difference
   goes to the project lead before anything changes.
6. Standard library only.
7. The port reproduces the recorded corpus exactly, upstream defects
   included. Deliberate fixes come after everything passes
   (`term51/docs/deliberate-fixes.md`).
8. SIMD fast paths are not ported; the scalar path is the reference.
9. The port never wraps errors (no `fmt.Errorf` with `%w`, no `errors.Join`):
   it returns the bare sentinel, so `err.Error()` is Zig's `@errorName`. Every
   sentinel is `var Err<ZigName> = errors.New("<ZigName>")`; the corpus
   replayer prints errors with `zigErrorName`, which panics for anything that
   is not one of the package's sentinels, and
   `TestCorpusReplaySelfErrorSentinels` checks every declaration.

## Names

| Zig | Go | Example |
|---|---|---|
| File-level struct `Foo.zig` | type `Foo` in package `terminal` | `Parser` |
| Declaration nested in a file or type, `Foo.Bar` | `FooBar` | `Parser.State` -> `ParserState`, `Parser.Action.CSI` -> `ParserActionCSI` |
| Enum value `.snake_case` | `<Type><CamelCase>` constant | `.csi_entry` -> `ParserStateCsiEntry` |
| `pub fn name` method | exported method `Name` | `next` -> `Next` |
| private `fn name` | unexported, name unchanged | `doAction`, `genTable`, `single` |
| `init` returning the type | function `<Type>Init` | `Parser.init()` -> `ParserInit()` |
| A name that is a Go keyword | trailing underscore | `range` -> `range_` |
| Struct fields | unchanged, including snake_case | `params_idx`, `param_acc` |
| File-level constants | unchanged | `MAX_PARAMS`, `ACCEPT_STATE` |

Every enum type has a `String()` method returning the Zig tag name, which is
what Zig's `@tagName` yields and what the recorded corpus contains.

## Language features

| Zig | Go | Notes |
|---|---|---|
| Optional `?T` stored or returned with other values | `Optional[T]` (`zig.go`) | `Valid` false is `null` |
| Optional pointer `?*T` | `*T`, nil is `null` | |
| Tagged union | struct with a `Tag` field and one field per payload | only the field named by `Tag` is meaningful; copies and comparisons behave as in Zig |
| Odd-width integers (`u21`, `u4`, ...) | next larger Go type | where Zig's width is observable (shifts that drop high bits) the Go code masks explicitly and says so |
| Saturating `*|`, `+|` | `saturatingMulU16`, `saturatingAddU16` (`zig.go`) | |
| `@addWithOverflow` on a counter that is allowed to wrap | plain Go addition on the same-width unsigned type | Go wraps identically |
| `assert(...)`, and every place Zig traps at run time (out-of-range index, integer overflow, `unreachable`) | `assert(cond)` (`zig.go`), which panics; `panic("unreachable")` | Go panics by itself on a bad slice or array index but not on a bit-set index, an overflow or a failed assertion. The port must stop wherever upstream would stop, so those are asserted explicitly, e.g. `StaticBitSet24.set` with an index of 24 or more. |
| `defer` | `defer` | |
| `errdefer` | not needed so far | to be fixed as one idiom when first met |
| `comptime` table | package-level `var` initialised once at start-up | `table = genTable()` |
| Function returning a type (`genTableType`) | the instantiations written out as named types | `Table`, `OptionalTable` |
| `inline for` over enum fields | loop over the enum's values | |
| `undefined` memory | Go zero value | never observable in upstream's behaviour |
| `std.StaticBitSet(24)` | `StaticBitSet24` (`zig.go`) | methods keep Zig's names |
| `log.warn(...)` | `warn(scope, ...)`, a no-op unless `logWarn` is set | |
| `std.fmt` `{f}` on a type with `format` | method `Format() string` | output matches Zig's so log lines stay comparable |
| `[]T` slices into parser-owned arrays | Go slices into the same arrays | same aliasing and same lifetime rule: valid until the next call |
| Allocator parameters | dropped | `deinit` functions are kept where they do something besides freeing |
| C ABI declarations (`C`, `cval`, `extern struct`) | not ported | the C bindings are out of scope |

## Per-file notes

### `parse_table.zig`
- `genTableType` is a type function; `Table` and `OptionalTable` are written
  out. Its two tests (dimension and element-type checks) translate to checks
  on the Go array types.
- `table` is computed at program start instead of at compile time. A test
  compares it entry by entry with upstream's table
  (`term51/golden/tables/parse_table.txt`).

### `UTF8Decoder.zig`
- Go's `unicode/utf8` is **not** a substitute: it emits one U+FFFD per
  invalid byte, upstream emits one per broken sequence (11,180 of 69,207
  recorded inputs differ). The decoder is ported.
- `accumulator` is a `u21` in Zig. The Go field is `uint32`, masked to 21
  bits after the shift so high bits are dropped exactly as in Zig.
- `next` returns `struct { ?u21, bool }`; Go returns
  `(Optional[uint32], bool)`.

### `Parser.zig`
- `next` returns `[3]?Action`; Go returns `[3]Optional[ParserAction]`.
- `osc_parser` is an `osc.Parser` (`osc.go`); its OSC paths needed no change
  when `osc.zig` was ported.

## Parser module: implementation notes

Recorded while porting the parser module (2026-10-05). Each part lists the Zig
features it maps and every decision a reviewer of that file will ask about.

### OSC core: `osc.zig`, `osc/encoding.zig`, `osc/parsers.zig`

#### Per-file notes: `osc.zig`

- `Parser.go` needed no change for the OSC port: its OSC paths (`osc_put`,
  the exit action of `osc_string` calling `end` and the entry action calling
  `reset`) already matched `Parser.zig`.
- `Parser.alloc` (`?Allocator`) is kept as `alloc any`: nil is null, any
  other value means "an allocator was provided". Nothing is allocated
  through it; it only decides what the parser accepts (`ensureAllocator`,
  `writeToAllocating`, the reset of a kitty colour command).
- `Parser.fixed` (`?std.Io.Writer`) and `Parser.allocating`
  (`?std.Io.Writer.Allocating`) are `Optional[...]`; `Parser.writer`
  (`?*std.Io.Writer`) is a Go pointer to `fixed.Value` or
  `allocating.Value.writer`, as in Zig (a copy of the parser after a writer
  was set up still points into the old copy, as in Zig).
- `std.Io.Writer.Allocating.initCapacity` cannot fail in Go; its `catch`
  branch in `writeToAllocating` (fall back to the fixed buffer) is kept
  but unreachable.
- `reset`: `v.deinit(self.alloc orelse break)` on a kitty colour command is
  `if self.alloc != nil { ...KittyColorProtocol.Deinit() }` (the allocator
  argument is dropped).
- Zig's exhaustive `switch` statements over `State` and `Command` get a
  `default: panic("unreachable")` (a value outside the enum traps in Zig).
- The Valgrind blocks of `init` and `reset` are not ported (they only mark
  `buffer` undefined); Go zeroes `buffer`.
- `ensureAllocator` logs with `warn("osc", ...)`; Zig's `{t}` of the state
  is `OSCParserState.String()`.
- `Terminator` keeps Zig's `string` as `String() []uint8`; the tag name is
  `TagName()`; `format` is `Format() string`, the terminator bytes.

#### `std.Io.Writer` and `std.Io.Writer.Allocating` stand-ins (`oscWriter`)

- `oscWriter` has Zig's `buffer` and `end` and one more field, `vtable`,
  which stands for Zig's vtable pointer (fixed or Allocating). Go cannot
  recover the Allocating that embeds a writer (`@fieldParentPtr`), so the
  writer itself says which drain it uses. Three values: `Fixed` (set by
  `oscWriterFixed`), `Allocating` (set by `oscWriterAllocatingInit`,
  `...InitCapacity` and on the first growth) and `Unset`, the zero value a
  composite literal gets: then a nil buffer means "an Allocating writer
  that has not allocated yet" and any other buffer means a fixed writer.
  This is what makes both test idioms behave as in Zig:
  `oscWriter{buffer: buf[:]}` is `.fixed(&buf)` and the zero
  `oscWriterAllocating` is `Allocating.init`. Limit: an
  `oscWriter{}` with a nil buffer is treated as an empty Allocating writer
  (in Zig `.fixed(&.{})` fails every write); use `oscWriterFixed(nil)` for
  that.
- Methods ported from std.Io.Writer with Zig's code and names: `write`,
  `writeAll`, `writeByte` (including the `while ... else` of writeByte),
  `buffered`, and `drain` dispatching to `oscWriterFixedDrain`
  (`Writer.fixedDrain`, including its partial copy before WriteFailed and
  its `assert(splat >= dest.len)`) or `oscWriterAllocatingDrain`
  (`Allocating.drain`). Others (`print`, `splatByteAll`, ...) are for the
  files that need them to add, same rule.
- `Allocating` methods: `init`, `initCapacity` (returns a nil error),
  `deinit` (drops the buffer; Zig leaves the value undefined),
  `toOwnedSlice` (copy of exactly the written bytes, writer left empty),
  `written`, `clearRetainingCapacity` (`end = 0`, Zig's
  `shrinkRetainingCapacity(0)`).
- Growth follows `ArrayList(u8).ensureUnusedCapacity` / `growCapacity`
  (`new +|= new / 2 + init_capacity`, init_capacity 128 = the 64-bit cache
  line). Not observable except through the buffer length; kept so the
  buffer always has the spare capacity the NUL-terminated (`[:0]`) results
  need, exactly when Zig has it. The `+|=` cannot saturate on any reachable
  size and is a plain `+=`.

#### `{any}` of an OSC command (`OSCCommand.zigFmtAny`)

- `Action.format` prints an `osc_dispatch` payload with
  `printValue("any", .{}, cmd, 3)`. `zigFmtAny` reproduces Zig 0.15.2's
  `printValue` for each variant at the depths it reaches: union at 3,
  payload at 2, payload fields at 1, their fields at 0 (struct ->
  `.{ ... }`, slice/array -> `{ ... }`; integers, bools, enums, null at any
  depth). Helpers: `zigFmtAnyStruct`, `zigFmtAnyBytes`, `zigFmtAnyElided`,
  `zigFmtAnyOptionalBool`.
- Two std containers show internals: `color.List` (`SegmentedList(Request,
  2)`) prints `.{ .prealloc_segment = { ... }, .dynamic_segments = { ... },
  .len = N }`; the kitty colour `std.ArrayList(Request)` prints
  `.{ .items = { ... }, .capacity = C }`. Go's slice capacity is not Zig's,
  so `oscKittyColorListCapacity(n)` computes Zig's: the parser starts from
  `.empty` and appends one at a time, so C is the first value of
  growCapacity's sequence with init_capacity = 128 / @sizeOf(Request) =
  128 / 12 = 10 that reaches n (0, 10, 25, 47, ...).
- Checked against upstream's `Action.format` output (built from the pinned
  source with the term51 Zig 0.15.2 toolchain) for every variant whose
  payload enums are already ported, and for the list capacities at 0, 2,
  11 and 26 requests. The type-name prefix is `terminal.Parser.Action`
  (Parser.go's existing choice), upstream prints
  `ghostty.terminal.Parser.Action`.

#### `osc/encoding.zig`

- `isSafeUtf8`: `std.unicode.Utf8View.init` is `utf8.Valid` (mapping rule
  5: a standard-library equivalent behind upstream's name), then a range
  over the string yields Utf8View's code points. Checked against the Zig
  function on all 16,777,216 three-byte strings and 16,777,216 four-byte
  strings (lead bytes F0-F7, a spread of continuation and non-continuation
  bytes): identical.

#### `osc/parsers.zig`

- Namespace file only; nothing to port (header updated).

### OSC sub-parsers: color, kitty_color, mouse_shape, change_window_icon, change_window_title, clipboard_operation, report_pwd, rxvt_extension, hyperlink

Files: `osc_parsers_color.go`, `osc_parsers_kitty_color.go`,
`osc_parsers_mouse_shape.go`, `osc_parsers_change_window_icon.go`,
`osc_parsers_change_window_title.go`, `osc_parsers_clipboard_operation.go`,
`osc_parsers_report_pwd.go`, `osc_parsers_rxvt_extension.go`,
`osc_parsers_hyperlink.go`.

#### Per-file notes: `osc/parsers/*.zig` (this group)

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

##### `osc/parsers/color.zig`
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

##### `osc/parsers/kitty_color.zig`
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

### OSC sub-parsers: osc9, semantic_prompt; `os/string_encoding.zig`

Files: `osc_parsers_osc9.go`, `osc_parsers_semantic_prompt.go`, and the new
`os_string_encoding.go` (see below).

#### Language features (additions)

| Zig | Go | Notes |
|---|---|---|
| Labeled block `blk: { ... break :blk; ... }` that is left from inside nested switches | Go block `{ ... goto blk ... }` followed by the label `blk:` | Go's `break` cannot leave a plain block, and from inside a nested `switch` it only leaves the switch. The label sits where `break :blk` lands; nothing is declared between a `goto` and its label. Used for `conemu:` (osc9.parse), `valid:` (semantic_prompt.parse), `value:` (Option.read) and `data:` (printfQDecode). |
| Labeled block that is a whole switch prong (`'A' => fresh_line: { ... break :fresh_line; ... }`, `progress: { ... }`) | `break` out of the Go `switch` | Equivalent because nothing follows the block inside the prong. The Zig label is kept as a comment. |
| `std.fmt.parseUnsigned` / `std.fmt.parseInt` with base 10, error discarded (`catch null`, `else \|_\|`) | a file-private port that returns `(value, ok)` | No Go function accepts the same inputs (Zig allows `_` between digits; parseUnsigned rejects a sign). Returning `ok` instead of an error avoids new sentinels (`ErrInvalidCharacter`, `ErrOverflow` would have to join `zigSentinels`); every caller discards the error. `oscParsersOsc9ParseUnsigned(buf, bits)` (u16 and usize) and `oscParsersSemanticPromptParseInt(buf)` (i32). subp-a has its own (`oscParsersColorParseIntU9`, ...); these could later be merged into one zig.go helper. |
| `usize` as the result type of a parse whose overflow is observable | 64 bits on every target | osc9 `9;4;1;<n>` parses into usize: on a 32-bit target Zig would reject 2^32..2^64-1, the recorded corpus (64-bit) clamps them to 100. The tests (`term51: parse: progress value`) expect the 64-bit behaviour, and pass under qemu-arm. |

#### Per-file notes

##### `osc/parsers/osc9.zig`
- `writer.writeByte(0) catch { ... }` is `if writer.writeByte(0) != nil`.
- `data[a .. data.len - 1 :0]` is `data[a : len(data)-1]`; the NUL just
  written stays in the capacity. The sentinel check always holds and is not
  asserted (same as subp-a).
- `.title = ""` (a `[:0]const u8` literal) is `[]uint8{0}[:0]`: empty, with
  the NUL in its capacity like Zig's literal.
- `std.math.clamp(x, 0, 100)` is `max(0, min(x, 100))`; the following
  `@intCast` to u8 is a plain conversion (the value is at most 100).
- `parser.command = .{ .x = ... }` assigns a whole `OSCCommand{Tag: ...,
  X: ...}`, so every other payload field is zeroed, as replacing the Zig
  union does.

##### `osc/parsers/semantic_prompt.zig`
- `Option.read` is comptime in `self`; the Go method switches on `self` at
  run time and fills the one field of `OSCParsersSemanticPromptOptionValue`
  named after it (wireframe decision). The `comptime self == .exit_code`
  test is an ordinary `if`.
- `Option.key` and the `.exit_code` prong of `read`'s final switch are
  `unreachable` in Zig: `panic("unreachable")`.
- Enum `String()` methods read name tables (`...Names`), as Parser.go does.
- `Command.init` sets `options_unvalidated` to `[]uint8("")` (Zig `""`).
- `writeCommandLine` calls the two decoders of `os/string_encoding.zig`,
  which no Go file had. The file is ported in `terminal/os_string_encoding.go`
  (next section): without them 17 translated tests (cmdline,
  cmdline_url, `term51: Command.writeCommandLine: ...`) cannot pass.

##### `os/string_encoding.zig` (new file, approved by the lead)
- File `terminal/os_string_encoding.go`, prefix `OsStringEncoding`: a file
  outside src/terminal named like `datastruct_circ_buf.go` (directory in
  the name). `pub fn`s are exported (`OsStringEncodingPrintfQDecode`,
  `OsStringEncodingUrlPercentDecode`, `OsStringEncodingUrlPercentEncode`),
  private ones take the lower-case prefix (`osStringEncodingHex`,
  `osStringEncodingIsValidChar`). The whole file is ported, in Zig order,
  although only the two decoders are used inside libghostty-vt.
- `hex` returns `u4` in Zig; the Go result is a uint8 that is always below
  16, and `std.math.shl(u8, hex(a), 4) | hex(b)` is `hex(a)<<4 | hex(b)` on
  uint8 (no bit is lost). Its `else => unreachable` is `panic("unreachable")`
  (not reachable: both callers check the digit first).
- `std.ascii.control_code.esc/lf/cr/ht/vt` are the literal bytes with the
  Zig name in a comment. `std.ascii.isPrint` is written out
  (`c < 0x80 && !(c <= 0x1f || c == 0x7f)`, std's isAscii and isControl).
- `std.Uri.Component.percentEncode(w, raw, comptime isValidChar)` is
  `osStringEncodingUriComponentPercentEncode`, a port of the std function
  with the comptime function parameter as a Go func value. Zig's
  `w.print("{s}%{X:0>2}", .{ raw[start..index], char })` writes the pending
  slice, `%` and two upper-case hex digits; the port writes the same bytes
  with three `writeAll` calls. Only with a fixed writer that runs out of
  room could the split of a partial write differ from std's formatter (not
  reachable in the library: urlPercentEncode has no caller in it).
- Tests: all 18 Zig tests (`printf_q 1`..`10`, `singles percent`,
  `percent 1`..`7`) are translated one for one in
  `terminal/os_string_encoding_test.go`.
- `scripts/check-test-mapping.py` needs a rule for `os/` (like the
  `datastruct/` one): `os/string_encoding.zig` currently resolves to
  `os/string_encoding_test.go` and reports all 18 tests missing. With
  `parts[0] in ("datastruct", "os")` and `base = parts[0] + "_" + ...` it
  reports `zig=18 go=18 ok` (checked on a temporary copy; the script is not
  edited).

### OSC sub-parsers: context_signal, kitty_clipboard_protocol, kitty_text_sizing, iterm2

Files: `osc_parsers_context_signal.go`, `osc_parsers_kitty_clipboard_protocol.go`,
`osc_parsers_kitty_text_sizing.go`, `osc_parsers_iterm2.go`
(`src/terminal/osc/parsers/*.zig`).

#### Language features (additions to the table)

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

#### Per-file notes

##### `osc/parsers/context_signal.zig`
- `Field.read` returns `OSCParsersContextSignalFieldValue`; the numeric arm
  (a labeled block in Zig) is inlined in `Read`, with no helper function.
- The empty-metadata literal `""` is `[]uint8("")` (non-nil, length 0).

##### `osc/parsers/kitty_clipboard_protocol.zig`
- `Option.read`'s `value:` block is a loop that either returns the null
  result or sets `value` and a `found` flag. The final `switch` fills the
  field named after `key`.
- `parse` keeps upstream's `assert(parser.state == .@"5522")`.

##### `osc/parsers/kitty_text_sizing.zig`
- `.{ .text = payload }` sets `scale: 1` explicitly (Zig's default).
- `update` parses the value before looking at the key, as upstream does, so an
  unknown key with a bad value reports `InvalidValue`.
- The two `it.next()` calls on the `'='` iterator are `it[0]` (always present:
  the "missing key" branch is kept but can't run, as upstream's can't)
  and `it[1]` (missing when the piece has no `'='`). Anything after a second
  `'='` is ignored, as upstream does.

##### `osc/parsers/iterm2.zig`
- `map` (Go `oscParsersIterm2Map_`) is built at package init by
  `oscParsersIterm2MapInitComptime`, the counterpart of `Map.initComptime` plus
  the comptime block that lists the fields of `Key`.
- Unknown and unimplemented keys set `parser.command = .invalid` and return
  null **without** setting `parser.state = .invalid` (upstream quirk, kept).
- The trailing `return &parser.command;` after the exhaustive `switch` is
  unreachable upstream and is kept.

#### New std stand-ins that could move to zig.go later
These are file-private now so they can't clash with other worktrees:
`oscParsersKittyTextSizingParseIntU4` (Zig `parseInt` for u4),
`oscParsersKittyClipboardProtocolIsWhitespace`,
`oscParsersKittyClipboardProtocolAsciiWhitespace`, `oscParsersIterm2AsciiLower`.
If other sub-parsers (osc9 etc.) also port `std.fmt.parseInt`, a shared
generic `zigParseInt` in `zig.go` could replace them after the merge.

### Colours and SGR: `sgr.zig`, `color.zig`, `x11_color.zig`, `kitty/color.zig`, `mouse_shape.zig`

Files: `sgr.go`, `color.go`, `x11_color.go`, `kitty_color.go`,
`mouse_shape.go`, plus `terminal/res/rgb.txt`. Sections below are ready to
merge into `docs/mapping.md`.

#### Per-file notes

##### `sgr.zig`
- `Parser.next`: the `underline:` labelled block. Go cannot `break` out of a
  labelled block; every `break :underline` leaves the block and then the
  `switch`, which is exactly what a plain Go `break` inside `case 4:` does
  (the code after the switch returns `.unknown`).
- `parseDirectColor` takes the tag as a run-time value; `@unionInit(Attribute,
  @tagName(tag), rgb)` is the helper `sgrAttributeInitRGB` (the three RGB
  variants; any other tag panics "unreachable", Zig's compile error).
- `countColon`: `self.params.len - 1` traps in Zig on an empty list; Go
  asserts `len(params) > 0` (never false: every caller has read a param).
- `@truncate` of a `u16` param to `u8` is the Go conversion `uint8(...)`.
- `@branchHint` has no Go counterpart; the comments next to them are kept.
- The separator list is `StaticBitSet24`, whose `isSet` asserts `index < 24`
  as Zig's does; the colon-skipping loop in `next` therefore stops the program
  where upstream would.

##### `color.zig`
- `color.default` and `DynamicPalette.default` are package variables
  initialised at start-up (`colorDefaultInit`, a function holding the comptime
  block; `colorDefaultCubeLevel` is the `if (c == 0) 0 else c*40+55`
  expression). The gray-ramp loop keeps `i +%= 1` on a `uint8` (wraps to 0).
- `std.StaticBitSet(256)` is `StaticBitSet256` (in `color.go`, to move to
  `zig.go`): four `uint64` masks, as Zig's `ArrayBitSet(usize, 256)`; `isSet`,
  `set`, `unset` assert `index < 256`. The iterator reads the live set at each
  step (Zig copies the current word); they differ only if the set changes
  during iteration, which no code does.
- `Name.String()` returns `""` for 16..255 as the wireframe (commit dde4530)
  says; dumps print `#N`. Note: mapping.md's section "Enums that are not
  exhaustive print `#N`" says `String()` should return `#N` itself. The dump
  output is the same either way (`dumpTagName` treats `""` as unnamed); the
  lead should pick one and fix the other text.
- `DynamicRGB.default` is the field `default_` (Go keyword rule).
- `RGB.fromIntensity` includes the term51 reference fix
  (`patches-reference/0002`, upstream-findings section 11): NaN returns
  `ErrInvalidFormat` (upstream aborts in `@intFromFloat`).
- `RGB.parse`: the `RGB{ .r = try fromHex(..), ... }` literals of the `#`
  forms are the helper `colorRGBFromHex3` (same order, first error wins).
- `fromHex`: `@intCast` to `u8` is an explicit `assert(v <= 255)`.
- `LAB.toRgb`: `@max`/`@min` on f32 return the non-NaN operand (LLVM
  maxnum/minnum); Go's built-ins return NaN, so `colorZigMaxF32` /
  `colorZigMinF32` are used. `@intFromFloat` truncates (Go conversion).

##### Floating point (all of `color.zig`)
- Fused multiply-add: Go may fuse `a*b + c` (it does on arm64); Zig/LLVM
  does not. Every product that is an operand of `+` or `-` is wrapped in an
  explicit `float32(...)`/`float64(...)` conversion (Go spec: an explicit
  conversion rounds and prevents fusion). Applied in `LAB.*`, `luminance`,
  `perceivedLuminance` and in every stand-in below. Go also fuses across
  statements, so a product stored in a variable that later feeds `+` or `-`
  is wrapped as well (`lo = float32(fk * ln2lo)`, `t1`, `t2`, `hfsq`, `z`,
  `w`, `x1 = float32(x1 * x1)`): to be safe every stored product in these
  functions is wrapped. Found by review after the parser module merged:
  the unwrapped stored products compiled to FMADD/FMSUB on arm64 and made
  `pow(x, 2.4)` 1 ULP off for some channels (e.g. 180); 32-bit ARM never
  fuses. `scripts/check-no-fma.sh` (run first by `scripts/test.sh`)
  disassembles the package for arm64 and amd64 v3 and fails on any fused
  instruction, since instruction selection can change with a Go upgrade.
- `std.math.pow(f32)`, `std.math.pow(f64)`, `std.math.cbrt(f32)`,
  `std.math.scalbn`/`ldexp` (f32, f64) are ported (`colorStdMathPowF32`,
  `colorStdMathPowF64`, `colorStdMathCbrtF32`, `colorStdMathLdexpF32/F64`).
  `@exp`/`@log` call compiler_rt's `expf`/`logf`/`exp`/`log` (musl-derived)
  in the recorder's build; those are ported too (`colorCompilerRtExpf`,
  `colorCompilerRtLogf`, `colorCompilerRtExp`, `colorCompilerRtLog`).
  Evidence: with `float32(math.Pow(float64(x), float64(y)))` instead of the
  port, `data/colors/lab_grid.txt` fails at line 9, `lab_lerp.txt` at line 2
  and `generate256.txt` fails. With the port all three match bit for bit,
  natively (arm64) and under 32-bit ARM qemu. (Go's `math.Cbrt` happened to
  match on the grid; the port is kept for fidelity.)
- `math.frexp` uses Go's `math.Frexp` on the value widened to float64
  (exact: same significand and exponent). `@trunc` (modf) uses `math.Trunc`
  (exact). `@sqrt` on f32 is `float32(math.Sqrt(float64(x)))`, the correctly
  rounded f32 root, as LLVM's.
- `std.math.nan(T)` is the quiet NaN `0x7FF8000000000000` / `0x7FC00000`
  (`colorStdMathNanF64/F32`); Go's `math.NaN()` has payload ...0001.
- Additional check run once during the port (not committed): Zig 0.15.2
  `std.math.pow(f64)` on 256x3 inputs and `pow(f32)`/`cbrt(f32)` on 2000x4
  inputs compared bit for bit with the Go stand-ins: identical natively and
  on 32-bit ARM.

##### `std.fmt.parseFloat(f64)` (used by `RGB.fromIntensity`)
- strconv's grammar is not Zig's: Zig accepts `_` between digits (also in the
  exponent) without a base prefix, a hex mantissa without `p` (`0x1`), and a
  signed `nan`; strconv rejects those. `colorStdFmtParseFloatF64` ports
  `parse_float.zig` + `parse.zig` (grammar, `Number`, `validUnderscores`,
  `parseInfOrNan`) and `convert_hex.zig` (hex conversion, including Zig's
  handling of more than 16 significant hex digits, which is not correctly
  rounded: `0x1.000000000000000800000000001p-1` gives 0.5,
  `0xFFFFFFFFFFFFFFFFFFFFp-80` gives 0x3FEFFFFFFFFFFFFE). Decimal values are
  converted by `strconv.ParseFloat` of the same digits without underscores:
  Zig's decimal path (fast path, Eisel-Lemire, slow path) is correctly rounded,
  as is strconv; `ErrRange` is accepted (it carries Zig's +-inf).
- The 8-digits-at-a-time fast path of `tryParseDigits` is not ported (same
  result, wrapping arithmetic).
- Checked once against Zig 0.15.2 on 60 edge inputs (signs, `.5`, `1.`,
  underscores valid and invalid, hex with/without `p`, inf/nan spellings,
  overflow/underflow, 20-50 digit decimals, >16-digit hex): identical bits
  and identical accept/reject.
- The error (`error.InvalidCharacter`) is discarded by the only caller, so
  the Go function returns `(float64, bool)`; no new sentinel.

##### `std.fmt.parseUnsigned` / `parseInt`
- `colorStdFmtParseUnsigned(buf, base, bits)` follows `parseIntWithSign`
  (`.pos`): no sign, `_` skipped but not first/last (consecutive `_` are
  accepted), accumulation in max(8, bits) bits with overflow as an error.
  Used by `RGB.fromHex` (u16, base 16), `Kind.parse` (u8, base 10) and
  `colorMap` (via `x11ColorParseIntU8`, which adds `parseInt`'s optional
  sign). Callers discard the error kind, so it returns `(uint64, bool)` and
  declares no `ErrInvalidCharacter`/`ErrOverflow` (avoids duplicates with the
  OSC port, which may declare them).
- `std.mem.trim` and `std.mem.indexOfScalarPos` are `colorStdMemTrim` and
  `colorStdMemIndexOfScalarPos`.
- All these stand-ins carry a `color` prefix and live at the end of
  `color.go` to avoid name clashes with stand-ins the parallel OSC port may
  add; they belong in `zig.go` and should be unified with the OSC port's
  equivalents when merging.

##### `x11_color.zig`
- `@embedFile("res/rgb.txt")` is `//go:embed res/rgb.txt` of a byte-for-byte
  copy of upstream's `src/terminal/res/rgb.txt` at `terminal/res/rgb.txt`
  (`TestX11Color_corpus_rgb_txt` compares it with the pinned upstream file).
- `map` (comptime `StaticStringMapWithEql(RGB, eqlAsciiIgnoreCase)`) is built
  at start-up by `colorMap`; a parse error (a compile error in Zig) panics.
  The Go map is keyed by the name with ASCII letters lower-cased
  (`x11ColorAsciiLower`); `get` lower-cases its argument the same way, which
  is `eqlAsciiIgnoreCase`'s comparison (non-ASCII bytes compare exactly). The
  782 names are distinct ignoring ASCII case, so no entry hides another.

##### `kitty/color.zig`
- `OSC.deinit` sets `list` to nil (`ArrayList.deinit` leaves it undefined).
- `Kind.parse`: `std.meta.stringToEnum(Special, key)` is an exact
  comparison with the tag names.
- `Kind.format` returns the decimal palette index or the tag name.

##### `mouse_shape.zig`
- `string_map` (`StaticStringMap`, exact match, distinct keys) is a Go map
  literal with upstream's 56 entries in upstream's order.
- `MouseShape.String` and the other exhaustive enums' `String()` index a
  names array; an out-of-range value (impossible in Zig) panics.


## Tests

- One Go test per Zig test, in the same order, with the exact Zig title in a
  `// Zig: test "<title>"` comment directly above it. `scripts/check-test-mapping.py`
  verifies this.
- Zig tests that are deliberately not translated are listed with their reason
  in `scripts/not-translated.txt` (C-binding helpers such as `cval`, tests of
  C ABI layouts, tests that only produce data files). Each needs the project
  lead's approval.
- Tests are translated before the implementation exists, against a wireframe
  (signatures only, every body panics "not implemented"). Until the
  implementation is approved those tests fail by design. A whole-package run
  aborts at the first not-implemented panic, so during the port the tests of
  ported files are run by name (`-run`); see `docs/port-status.md`.
- Tests that read recorded data (`corpus_color_test.go`, `corpus_test.go`) use
  the `term51` checkout named by `LIBGHOSTTY_VT_CORPUS`.

## Wireframe naming decisions (project lead)

- A Zig file in a subdirectory is part of the same Go package, with the
  subdirectory in the name: `kitty/color.zig` is `kitty_color.go`, type prefix
  `KittyColor`; `osc/parsers/*.zig` follow the same rule.
- Namespace files: `color.RGB` is `ColorRGB`, `color.default` is
  `ColorDefault`, `x11_color.map` is `X11ColorMap`.
- SGR: tag constants are `SGRAttributeTag<Variant>`; struct fields for
  variants whose name starts with a digit get an `X` prefix (`X8Bg`,
  `X256Fg`, ...); `usize` is `uint`; `Parser.empty` is `SGRParserEmpty`.
- Floating point: the LAB code is 32-bit float arithmetic. Go may fuse a
  multiply and an add into one instruction on ARM, which Zig does not do and
  which changes the last bit. When implemented, wrap each product in an
  explicit `float32(...)`; `data/colors/*.txt` in the corpus catches any
  difference.

## Requirements the wireframe tests impose on the implementation

(Accepted by the project lead; recorded so they are not lost.)

- Strings that Zig returns as NUL-terminated slices (`[:0]const u8`) must be
  sub-slices of the parser's buffer with spare capacity covering the
  terminating zero byte: about eight OSC tests read the byte after the slice
  (`s[:len+1][len]`).
- A null result is the zero value of `Optional[T]`.
- `oscWriter`/`oscWriterAllocating` behave like Zig's `std.Io.Writer` fixed
  and allocating writers: a zero-value allocating writer grows on demand.
- The integrity check gets the same deferral as the corpus's candidate B at one
  place (inside `resize`, the "viewport pin has enough rows" check), with a
  comment pointing at `docs/upstream-findings.md` and
  `docs/deliberate-fixes.md` entry 3. No behaviour fix during the port.
- `Stream.nextSliceCapped` must not abort on input such as `FF ESC [ 1 A`: it
  returns what `nextSlice` returns for the same bytes (`parser/nextslicecapped_inputs`).

## Naming rules settled with the small-file wireframes (project lead)

- A declaration with the same name as its file is not doubled: `Style`,
  `Point`, like `MouseShape`; so `point.Coordinate` is `PointCoordinate`.
- Private file-level constants and tables take the file prefix, like private
  functions: `tabstopsUnitBits`, `tabstopsPreallocColumns`,
  `tabstopsPreallocCount`, `tabstopsMasks`.
- `Tabstops.Deinit` is kept as a stub because every test calls it.
- `Stream[H]` holds its handler by pointer. A handler "passed by value" in
  Zig therefore cannot be told apart from one passed by pointer in Go; two
  tests that check this are translated with a pointer handler and lose that
  distinction (a Go limitation, accepted).
- The reference-counted set is generic over all four of Zig's parameters, has
  no memory base, keeps `cap`, `table_cap` and `table_mask` as counts (the
  byte offsets are dropped), treats a zero-capacity set as empty (lookups find
  nothing, adds return `ErrOutOfMemory`), and uses integer arithmetic that must
  reproduce `data/sets/capacities.txt` (every capacity 0 to 65,536).
- Hashes (`Style.hash`, `PageEntry.hash`) must reproduce
  `data/sets/style_hash.txt` and `data/sets/hyperlink_hash.txt`: they decide
  when a set reports that it is full and so drive page growth. `Style.hash`
  needs Zig's `std.hash.int`; `PageEntry.hash` needs Wyhash with `autoHash`
  semantics, neither of which the Go standard library has.

## Page-layer bookkeeping (decision: option A, final)

Confirmed by the project owner on 2026-10-04. The port keeps upstream's byte
bookkeeping on ordinary Go slices: `BitmapAllocator` (chunk bitmaps and byte
layout), `hash_map` (layout, control bytes, hashing, probing), `Page.layout`,
the page-list size accounting (`page_size`, `maxSize`, pool item size). The
scrollback limit is counted in bytes of page memory as in Ghostty, so the
same output keeps the same history. All byte sizes and offsets are `uint64`
(the recorded data come from a 64-bit build; the phone is 32-bit ARM). Tests
that depend on rows per page or literal byte sizes set
`PageStdCapacity.grapheme_bytes = 512` through the shared `testStdCapacity`
helper (`terminal/zig_defaults_test.go`) and never run in parallel; the
recorded-corpus tests leave the production value (8192).

`PageList.diagram` and `HighlightFlattenedInit` are not ported (upstream's
bodies cannot compile; they have no behaviour). `Screen.clone` with the cursor
outside the cloned region returns `ErrScreenCloneCursorOutside` instead of
upstream's abort.

## Platform branches in the input encoders (decision: platform-neutral, final)

The Go library is platform neutral (project owner, 2026-10-04): it follows
upstream's compile-time platform checks at run time. Upstream has two checks in
`src/input/` (`key_encode.zig`, `key_mods.zig`, `key.zig`): `builtin.os.tag ==
.macos` and `builtin.target.os.tag.isDarwin()`. In Go they read `platformGOOS`
(`terminal/platform.go`; the implementation sets it from `runtime.GOOS`):
`platformIsMacos()` is GOOS "darwin"; `platformIsDarwin()` is "darwin" or "ios".
There are three modes: linux (both false: everything that is not Apple), darwin
(both true) and ios (Apple but not macOS: `isDarwin()` true, `.macos` false; the
one case where the distinction changes output: modifyOtherKeys drops alt but
the ESC prefix ignores `macos_option_as_alt`).

Testing. Upstream cannot be run on a Mac here, so, like the other reference
patches, `patches-macos/` and `patches-ios/` build upstream with the checks forced
(`scripts/build-macos.sh`), the whole suite runs in each mode, and the keyboard
corpus is recorded per mode (`golden/input`, `golden-macos/input`,
`golden-ios/input`; `docs/macos.md` in the term51 repository). In Go every encoder
test selects its platform with the shared helper `testPlatform` (sets and
restores `platformGOOS`, no parallel tests): non-platform tests run as "linux";
tests Zig gates on `isDarwin()` run under "darwin" and "ios", those gated on
`.macos` under "darwin"; shared tests whose expected values depend on the platform
loop over the three modes (`// Go-only: platform modes`). Tvos, watchos, visionos and
DriverKit are not Go targets and are not modelled.

## Enums that are not exhaustive print `#N`

Every `String()` of an enum that Zig declares non-exhaustive (`ColorName`, the ANSI C0
enum, mode and key enums with an `_` arm, and so on) returns `""` for a value
that has no name, and the dumper prints such a value as `#N` (decimal), as the
recorder's `dumpAny` does (project lead, 2026-10-05). The replayer
(`terminal/corpus_replay*_test.go`) depends on it. Sentinel errors carry Zig's
error name as their text for the same reason.
