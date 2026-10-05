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
   Ruling (project lead, 2026-10-05): `Stream.nextSliceCapped` calls
   `simd.vt.utf8DecodeUntilControlSeq`, whose scalar fallback
   (`utf8DecodeUntilControlSeqScalar`) lives in the SIMD module and is only
   reached from the SIMD branch of `nextSlice`, so it is not ported. Its
   defined result is what per-byte `next` gives (upstream's fallback aborts
   on `FF ESC [ 1 A` and disagrees with `next` on `E2 82 (`, findings
   section 12). `stream.go` uses the stand-in `streamUtf8DecodeUntilControlSeq`
   (see "`stream.zig`" below), and
   `TestStream_go_only_nextSliceCapped_equals_per_byte_next` enforces the
   equivalence on 30,000 seeded random inputs fed in random chunks (printed
   codepoints, action log, parser and UTF-8 decoder state).
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
- Tests: all 18 upstream tests (`printf_q 1`..`10`, `singles percent`,
  `percent 1`..`7`) and the 32 term51 tests (patch 0020, one or more for
  every function, added at the lead's request) are translated one for one
  in `terminal/os_string_encoding_test.go`. The library's Zig test root
  never reached this file, so upstream's 18 had never run in our Zig build;
  patch 0020 adds `test { _ = string_encoding; }` to `semantic_prompt.zig`
  (listed in `scripts/not-translated.txt`).
- `scripts/check-test-mapping.py` maps `os/` like `datastruct/`
  (`os/string_encoding.zig` -> `os_string_encoding_test.go`).

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
- Ported in full from Zig 0.15.2 (`lib/std/fmt/parse_float.zig` and
  `parse_float/*.zig`) as `terminal/std_fmt_parse_float.go`, entry point
  `stdFmtParseFloatF64`; no `strconv`. strconv differs from Zig in grammar
  (`_` separators, `0x1` without `p`, signed `nan`), in hex rounding (Zig is
  not correctly rounded with more than 16 significant hex digits), in decimal
  rounding (Zig's slow path, `Decimal.parse`, drops digits when the input
  has `_` separators and returns a value one unit in the last place low:
  `rgbi:0.0_66666666666666658802586908905141171999275684356689453125001/0/0`
  gives red 16, strconv 17; term51 `docs/upstream-findings.md` section 21,
  `docs/deliberate-fixes.md` entry 7), and in exponents (strconv caps them
  at 10,000). The port reproduces all of it. Details, names and the
  bit-for-bit comparison with Zig: "`std.fmt.parseFloat(f64)`:
  `terminal/std_fmt_parse_float.go`" below.
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
- All these stand-ins (except `std.fmt.parseFloat`, which is in
  `std_fmt_parse_float.go` with the prefix `stdFmtParseFloat`) carry a
  `color` prefix and live at the end of `color.go` to avoid name clashes with stand-ins the parallel OSC port may
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


### `std.fmt.parseFloat(f64)`: `terminal/std_fmt_parse_float.go`

Zig 0.15.2's `lib/std/fmt/parse_float.zig` and `lib/std/fmt/parse_float/*.zig`
(the toolchain that built the recorded corpus) are ported in full. Nothing in
the file is ported from a Ghostty source file. The only caller is
`RGB.fromIntensity` (`colorRGBFromIntensity`, `color.go`), which parses an
f64.

#### Why a port and not strconv

- Grammar: Zig accepts `_` between digits (also in the exponent) without a
  base prefix, a hex mantissa without a `p` exponent (`0x1`) and a signed
  `nan`; strconv rejects those.
- Hex rounding: with more than 16 significant hex digits Zig is not correctly
  rounded (`0x1.000000000000000800000000001p-1` gives 0.5,
  `0xFFFFFFFFFFFFFFFFFFFFp-80` gives 0x3FEFFFFFFFFFFFFE).
- Decimal rounding: Zig's decimal path is **not** always correctly rounded.
  `Decimal.parse` (`decimal.zig` lines 273-294, used by `convertSlow`) trims
  trailing zeros by scanning `s` backwards from `stream.offsetTrue() - 1`.
  `offsetTrue()` does not count the `_` separators that `scanDigit` skipped,
  so with separators the scan starts too far left. If it lands on a run of
  `'0'` characters it counts them, and `num_digits -= n_trailing_zeros` then
  drops that many digits from the end, real digits included. Near a
  midpoint between two doubles this returns the lower one (one unit in the
  last place low). Example: `0.0_66666666666666658802586908905141171999275684356689453125001`
  gives 0x3FB1111111111110; the correctly rounded value (strconv, and Zig
  without the `_`) is 0x3FB1111111111111. Through `rgbi:` that is red 16
  instead of 17. Only inputs that reach the slow path are affected, which
  means more than 19 significant digits near a midpoint, with Eisel-Lemire
  unable to decide.
- Exponents: `Decimal.parse` saturates its exponent at 0x10000 (65,536) and
  `parseScientific` at 0x1000_0000. strconv caps the exponent at 10,000, so
  long inputs with compensating exponents of 100,000 or more differ.

The port copies all of these exactly (governing rule 7).

#### Names

- Every declaration has the prefix `stdFmtParseFloat` followed by its Zig
  name in Go style: `parseFloat` -> `stdFmtParseFloatF64`,
  `FloatStream` -> `stdFmtParseFloatFloatStream`, `isEightDigits` ->
  `stdFmtParseFloatIsEightDigits`, file-level constants
  (`max_shift`, `eisel_lemire_smallest_power_of_five`, ...) ->
  `stdFmtParseFloatMaxShift`, `stdFmtParseFloatEiselLemireSmallestPowerOfFive`.
  The prefix names the std file group; unlike the earlier stand-ins
  (`colorStd...`), it carries no `color` prefix: the code is no longer in
  `color.go`, and the name says what it ports, not who calls it.
- Generic instantiations get a type suffix. Generic over the float type T
  (always f64 here): `F64` (`stdFmtParseFloatConvertFastF64`,
  `stdFmtParseFloatNumberF64`, `stdFmtParseFloatBiasedFpF64`,
  `stdFmtParseFloatDecimalF64`, ...). Generic over the mantissa type
  (`mantissaType(f64)` is u64): `U64` (`stdFmtParseFloatTryParseDigitsU64`,
  `stdFmtParseFloatMinNDigitIntU64`, `stdFmtParseFloatFastIntPow10U64`).
- Methods keep Zig's names in lower case (`offsetTrue`, `scanDigit`,
  `leftShift`, `toFloat`, ...): the types are file-private. Functions without
  `self` that return the type follow the `<Type>Init` rule:
  `FloatStream.init` -> `stdFmtParseFloatFloatStreamInit`,
  `BiasedFp.zero`/`zeroPow2`/`inf` -> `stdFmtParseFloatBiasedFpF64Zero` /
  `...ZeroPow2` / `...Inf`, `Decimal.new`/`parse` ->
  `stdFmtParseFloatDecimalF64New` / `...Parse`, `U128.new`/`mul` ->
  `stdFmtParseFloatU128New` / `...Mul`.
- `FloatStream.len` is the method `len` (legal in Go). The local `len` of
  `parsePartialNumberBase` is `len_`.
- std helpers used by the package: `std.ascii.toLower` ->
  `stdFmtParseFloatAsciiToLower`, `std.ascii.startsWithIgnoreCase` ->
  `stdFmtParseFloatAsciiStartsWithIgnoreCase`, `std.math.shl(u64, ...)` /
  `std.math.shr(u64, ...)` -> `stdFmtParseFloatMathShlU64` /
  `stdFmtParseFloatMathShrU64` (signed shift amount: negative reverses the
  direction, 64 or more gives 0, as Zig's), `std.math.nan(f64)` ->
  the existing `colorStdMathNanF64` (`color.go`), `std.math.inf` ->
  `math.Inf`, `std.math.mul(u64)` -> `bits.Mul64` with a non-zero high word
  as the overflow, `@clz` -> `bits.LeadingZeros64`, `std.mem.readInt` /
  `writeInt` (u64, little) -> `encoding/binary.LittleEndian`. All could move
  to `zig.go` later.

#### Layout

One Go file for the ten Zig files (the task asked for one file). Each Zig
file is a section headed by its path, functions in the Zig order: 
`parse_float.zig`, `parse.zig`, `FloatStream.zig`, `common.zig`,
`FloatInfo.zig`, `convert_hex.zig`, `convert_fast.zig`,
`convert_eisel_lemire.zig`, `convert_slow.zig`, `decimal.zig`. Upstream's
comments are carried over.

#### Comptime

- Only the f64 instantiation exists. The f16, f32, f80 and f128 arms of
  every type switch (`FloatInfo.from`, `fastPow10`, `floatFromUnsigned`,
  `BiasedFp.inf`) and the u128 arm of `fastIntPow10` are not ported;
  `if (MantissaT == u64) a else b` is written as `a` (max_mantissa_digits 19
  and 16, Decimal constants 768/19/2047/-324/310/18). `mantissaType(T)` is
  `uint64` wherever it is used.
- `FloatInfo.from(f64)` (comptime_int fields) is a block of untyped Go
  constants `stdFmtParseFloatFloatInfoF64<Field>`.
- Comptime value parameters become ordinary parameters: `base`, the
  `ParseInfo` struct (`stdFmtParseFloatParseInfo`), `precision` of
  `computeProductApprox`, `n` of `tryParseNDigits`. `comptime assert`s on
  them are run-time `assert`s.
- `math.floatExponentBits(f64)` etc. are literal constants (11, 52, 1023,
  -1022) with the Zig expression in a comment.
- Tables: `eisel_lemire_table_powers_of_five_128` (651 entries) and
  `pow2_to_pow5_table` (128 entries, local to `numberOfDigitsLeftShift` in
  Zig, package-level in Go: `stdFmtParseFloatDecimalPow2ToPow5Table`) were
  copied from the Zig source by a script and compared entry by entry with
  it (identical; every cutoff is also checked to equal 5^i). Note that the
  Eisel-Lemire table passes the high word of each power as `U128.lo` and the
  low word as `.hi`; `computeProductApprox` uses them that way round, and so
  does the port. `fastPow10` and `fastIntPow10` tables are package-level
  arrays (`...Table`) indexed by the function as in Zig (`i & 31`).

#### Optionals and errors

- `?T` results are `(T, bool)` (as the previous stand-ins did):
  `FloatStream.first`, `readU64`, `scanDigit`, `parseScientific`,
  `parsePartialNumberBase`, `parsePartialNumber`, `parseNumber`,
  `parsePartialInfOrNan`, `parseInfOrNan`, `convertFast`,
  `convertEiselLemire`.
- `parseFloat`'s only error, `error.InvalidCharacter`, is discarded by the
  only caller, so `stdFmtParseFloatF64` returns `(float64, bool)`; no new
  sentinel.

#### Control flow

- `scanDigit`'s `retry: while (true)` with `continue :retry` is a Go `for`
  with `continue`.
- `convertSlow`: the `blk:` block whose `5...9` prong does an unlabelled
  `break` (which leaves the enclosing `while`) is a labelled Go loop
  `shiftLeft:` with `break shiftLeft`.
- `numberOfDigitsLeftShift`: the `return x.delta;` after the if/else chain
  inside the `for` is unreachable in Zig and is omitted (`go vet` rejects
  unreachable code); a comment marks the place.
- `@branchHint(.cold)` has no Go counterpart (comment kept).

#### Safety checks (ReleaseSafe recorder)

Where Zig would trap and Go would wrap silently, the port asserts:
`n.mantissa + 1` in `parseFloat`, `n *= 10` in `min_n_digit_int`,
`first.hi += 1` in `computeProductApprox`, `@intCast(self.e)` to u64 in
`BiasedFp.toFloat` (e >= 0), the `@intCast` usize -> i32 conversions and the
`i -= 1` and `d.num_digits -= n_trailing_zeros` subtractions in
`Decimal.parse`. None of them can fail for any input (reasoning in the
comments; the 11.6 million compared inputs never trip them, natively or on 32-bit ARM). Other checked
operations are proven not to overflow by upstream's own comments
(`parse8Digits`) or by bounds (shift amounts, `Decimal` digit arithmetic in
u64 with shifts of at most 60); Go's index checks cover the array accesses.

#### Integer widths

`usize` is `uint`, `isize` is `int64` (the recorder is 64-bit; the values are
bounded by the input length), `i64`/`i32`/`u64`/`u8` are the same-width Go
types. `U128.mul` is `bits.Mul64`. Nothing uses Go `int` where the width
matters, so 32-bit ARM computes the same values (checked, below).

#### Floating point

The only float arithmetic is in `convertFast`: `float64(mantissa)` (exact,
mantissa <= 2^53), one multiplication or division by a power of ten, and a
negation. There is no addition, so Go cannot fuse anything; `go build
-gcflags=...=-S` of the package shows no FMADD/FMSUB/FNMADD/FNMSUB.
Eisel-Lemire and the slow path are integer-only.

#### Not ported as an optimisation

Nothing is left out: the 8-digits-at-a-time path of `tryParseDigits`
(`parse8Digits`, `isEightDigits`, `readU64Unchecked`) and the 8-digit copy
in `Decimal.parse` are ported (the previous stand-in skipped the first one).
Unused declarations (`FloatStream.len`, `isEmpty`, `readU64`, `atUnchecked`,
`BiasedFp.zeroPow2`) are ported too.

#### Verification

Compared bit for bit (f64 bits, or ERR) with Zig 0.15.2 `std.fmt.parseFloat(f64)`
using the project lead's harness (copy in `/home/xia/wt/pf-scratch/pfdiff-pf`,
streamed, nothing stored):

| Family | What | Inputs | Zig ERR | Diffs native (arm64) | Diffs 32-bit ARM (qemu) |
|---|---|---:|---:|---:|---:|
| targeted | hand-written edge cases (signs, `inf`/`nan` spellings, hex with/without `p`, underscores valid/invalid, under/overflow) | 123 | 61 | 0 | 0 |
| rgb | the 255 channel boundaries with and without `_`, `rgbi:` domain | 510 | 0 | 0 | 0 |
| bytes | every byte value inserted/substituted at every position of 8 templates | 14,336 | 13,833 | 0 | 0 |
| long | very long inputs (up to 200,000 digits, with and without `_`, decimal and hex) with compensating exponents | 297 | 0 | 0 | 0 |
| intensity | random values in [0, 1] with long digit strings | 200,000 | 63,949 | 0 | 0 |
| random | random decimal and hex strings, random `_` | 300,000 | 24,124 | 0 | 0 |
| halfway | exact midpoints between adjacent doubles, perturbed and truncated, with and without `_` | 1,394,822 | 412,252 | 0 | 0 |
| extra | midpoints with `_` (>19 digits) at the intensity boundaries, huge exponents, hex, inf/nan, malformed | 1,639,789 | 209,253 | 0 | 0 |
| exh6 | every 6-character string over three 11/8-letter alphabets | 3,805,266 | 3,795,432 | 0 | 0 |
| exh5 | every string of up to 5 characters over a 21-letter alphabet | 4,288,305 | 4,275,771 | 0 | 0 |
| **total** | | **11,643,449** | | **0** | **0** |

The Go side is the current `std_fmt_parse_float.go` (package line changed
to `main`), built natively and with `GOARCH=arm GOARM=7`, run under
`qemu-arm-static`; the whole set ran on both. Defect-focused check
(`/home/xia/wt/pf-scratch/defect`): `RGB.parse` built from upstream
`color.zig` with Zig 0.15.2 against `terminal.ColorRGBParse`, natively and
on 32-bit ARM: identical on the 10 `rgbi:` strings of `list.txt` and on all
1,020 generated candidates (`cands.txt`: for each channel boundary k, the
midpoint + `001` with `_` after `0.0`, after `0.0` and two digits later,
near the end, and without `_`); 360 of the 765 underscored candidates give
a channel one lower than the correctly rounded value, none of the 255
without `_`.

Tests: `terminal/std_fmt_parse_float_test.go` translates the six tests of
`parse_float.zig` that have an f64 part (`parseFloat`, `nan and inf`,
`largest normals` (f64 line), `hex.special` and `hex.zero` (written for f32
in Zig, width-independent, run on f64), `hex.f64`); not translated:
`#11169` (f128), `many_digits hex` (f32 vs f128), `hex.f16`, `hex.f32`,
`hex.f80`, `hex.f128`. `parse_float/*.zig` have no tests.
`scripts/check-test-mapping.py` does not check std files (as for
`std_hash_*_test.go`). Two Go-only tests pin the defect bits and the large
exponents. The defect is also pinned through `rgbi:` by the term51 test
`term51: RGB.fromIntensity: underscores can lower a channel by one (zig std
defect)` (term51 patch 0022, translated in `color_test.go`) and by the
recorded suite `osc/rgbi_float` (57 cases: OSC 11, OSC 4 and OSC 21 with
each string, with and without `_`).

## Stream module: implementation notes

Recorded by the implementers of each file and folded in here. Governing
rules are above; these sections only add what the implementation decided.

### Small terminal files: `ansi.zig`, `charsets.zig`, `modes.zig`, `device_status.zig`, `kitty/key.zig`

Files: `ansi.go`, `charsets.go`, `modes.go`, `device_status.go`,
`kitty_key.go`. Recorded while porting them (2026-10-05).

#### Language features (additions to the table)

| Zig | Go | Notes |
|---|---|---|
| `packed struct(uN)` whose bit layout is observable (`ModeTag`, `device_status.Tag`, `kitty.key.Flags`) | ordinary Go struct plus the wireframe's conversion pair (`Int()` = `@bitCast` to the backing integer, `...FromInt` = `@bitCast` back) | the conversions place each field at Zig's bit offset (first field = bit 0). A field narrower than its Go type (`u15` value) and a `u5` argument are asserted to fit (`assert(value <= 0x7FFF)`, `assert(v <= 0x1F)`): such values cannot exist in Zig, so the port stops instead of dropping bits. The reverse cast from the full backing integer (`ModesModeTagFromInt`) needs no check: every u16 is a valid ModeTag. |
| `packed struct` used only as a set of bools (`ModePacked`) | ordinary Go struct of bools, fields in Zig's order | no Go code depends on the bit layout; the tests check the field order with reflect (the counterpart of `@bitOffsetOf`) |
| Type generated at compile time from a table (`Mode`, `ModePacked`, `device_status.Request`, via `@Type`) | the declarations written out (wireframe), plus the part of the generation that is data computed from the table at start-up | `Mode`/`Request`: the enum field list (name, value = the Tag bit-cast) is built by `modesModeInit` / `deviceStatusRequestInit` from `entries` into `modesModeFields` / `deviceStatusRequestFields`; `String()` and `ModesModeValues()` read it. The written-out constants equal what it computes (the tests check all of them). `ModePacked`'s field defaults (`.default_value_ptr = &entry.default`) are set by `ModesModePackedInit` in a loop over `entries`. |
| `@field(s, name)` with a comptime field name (`modes.zig`) | `modesModePackedField(p, name) *bool`, a `switch` on the name returning a pointer to that field | an unknown name is a compile error in Zig; it panics in Go (cannot happen: every entry has a field). `132_column` maps to the Go field `x132_column`. |
| `switch (mode) { inline else => \|m\| { const entry = comptime entryForMode(m); ... @field(..., entry.name) } }` | tables built once at start-up: `modesModeIndex` (Mode -> entry index, a [1<<16]uint8) and `modesModePackedFields` (per entry, the field accessor `@field` resolves to, checked against `modesModePackedField` at start-up); set/get/save/restore are one index load plus one field access | rule (lead, 2026-10-05): compile-time work is computed once at start-up, never per call. The first implementation did the lookup per call (`String()`, a scan of `entries`, a string switch; ~420 ns per `get`); now ~9 ns. `modesEntryForMode` and `modesModePackedField` keep their Zig meaning but are off the per-call path. A value that is not a Mode panics (`modesModeEntryIndex`; Zig: switch on a corrupt enum value traps). |
| `inline for (entries)` with `comptime` conditions (`modeFromInt`, `reqFromInt`) | `modeFromInt`: lookup of the tag `(v, ansi)` in `modesModeIndex`, then the `disabled` check; `reqFromInt`: ordinary `for` loop and `if` | Zig unrolls the loop into one integer comparison per entry. `reqFromInt` (3 entries) keeps the loop; `modeFromInt` uses the start-up index at the lead's request. |
| `@splat(.disabled)` (FlagStack) | loop assigning `KittyKeyFlagsDisabled` to every element | |
| `u3` counter with `+%=` / `-%=` (`FlagStack.idx`) | uint8, `(idx + 1) & 7` / `(idx - 1) & 7` | wraps exactly as the u3 |
| `~` on a `u5` (`FlagStack.set`, `.not`) | `^` on the uint8 | sets bits 5..7 too, but the `&` with the current flags (at most 31) clears them, so the result is the u5 result |
| `pub inline fn` (`charsets.table`) | ordinary function | inlining left to the Go compiler |

#### Enum `String()` methods

- Exhaustive enums (all `lib.Enum`s, `ProtectedMode`, `Slots`, `ActiveSlot`,
  `Charset`, `SetMode`) read a file-private names array (`ansiCursorStyleNames`,
  `charsetsSlotsNames`, `kittyKeySetModeNames`, ...); an out-of-range value
  panics, as in `mouse_shape.go` / `Parser.go`.
- Non-exhaustive enums (`ANSIC0`, `ANSIRenditionAspect`, `ANSIStatusLineType`)
  use a `switch` and return `""` for an unnamed value (dumps print `#N`), per
  "Enums that are not exhaustive print `#N`".
- Generated enums (`ModesMode`, `DeviceStatusRequest`) search the generated
  field list and panic("unreachable") for a value outside the enum.

#### Per-file notes

##### `ansi.zig`
- No functions; only the String() methods. Upstream's inline comments (the
  `// Blank`, `// >`, `// =` of DeviceAttributeReq, the "Non-exhaustive so
  that ..." comments) are carried over.

##### `charsets.zig`
- `ascii`, `british`, `dec_special` are package variables initialised at
  start-up; the `british:` and `tech:` blocks are function literals called
  in the initialiser.
- `table`: `.utf8 => unreachable` is `panic("unreachable")`, and a value
  outside the enum also panics (exhaustive switch). The returned slice
  aliases the package array, as Zig's points at the comptime constant (the
  storage-identity test depends on that); callers must not write through it.
- Upstream's doc comment of `table` says the slice has 255 chars; it has
  `table_len` = 256. The comment is carried over with a note.
- `initTable` keeps the `while` loop and `assert(i == table_len)`.

##### `modes.zig`
- `ModeState{}` / `ModePacked{}` are `ModesModeStateInit()` /
  `ModesModePackedInit()` (wireframe). A zero `ModesModeState` is NOT Zig's
  default state (six modes default to true).
- `reset`: `self.saved = .{}` is `ModesModePackedInit()` (the type's defaults,
  not `self.default`), as upstream.
- `get` takes `*const ModeState` in Zig; Go uses the pointer receiver.
- `entryForMode`: `std.mem.eql(u8, entry.name, @tagName(mode))` is
  `string(entry.name) == mode.String()`; `@setEvalBranchQuota` has no
  counterpart; the final `unreachable` is `panic("unreachable")`.
- `ModesModeValues` (std.enums.values) returns a new slice per call.
- `golden/tables/modes.txt` (TestCorpusReplayTables/modes) passes.

##### `device_status.zig`
- Same scheme as modes.zig for the generated `Request` enum and `Tag`.
- The wireframe has no reverse `@bitCast` for `Tag` (modes has
  `ModesModeTagFromInt`), so three lines of
  `term51: Tag: layout of the value and question bit` stay untranslated
  (marked MISSING-IN-WIREFRAME in the test). A `DeviceStatusTagFromInt`
  identical in shape to `ModesModeTagFromInt` would make them translatable;
  not added here (signatures are the lead's).

##### `kitty/key.zig`
- `Flags.int` builds the u5 from the five bools (bit 0 = disambiguate ...
  bit 4 = report_associated); `KittyKeyFlagsFromInt` is the reverse and
  asserts `v <= 31`. stream.zig only casts values already checked with
  `std.math.cast(u5, ...)`, so the assertion never fires from the stream.
- `FlagStack.set`: the exhaustive switch gets `default: panic("unreachable")`.
- `FlagStack.pop(n)`: `n` is `uint` (usize); `n >= self.flags.len` resets the
  stack exactly as upstream (idx = 0, every slot disabled).

### `stream.zig`

File: `terminal/stream.go`. Recorded while porting it (2026-10-05). The design of the generic, the handler interface
and `Action` is the wireframe's (header of `stream.go`); unchanged.

#### Language features (additions)

| Zig | Go | Notes |
|---|---|---|
| `try self.handler.vt(.key, payload)` | `if err := self.handler.Vt(StreamActionKeyKey, StreamAction{Tag: StreamActionKeyKey, Key: payload}); err != nil { return err }` | the union is built in place at every call; no helper. A void payload is `StreamAction{Tag: ...}` |
| `@hasDecl(T, "vtRaw")` (comptime) | `any(h).(StreamHandlerVtRaw)` once in `StreamInit`, kept in the added field `vt_raw` (nil: no vtRaw); `nextNonUtf8` tests `self.vt_raw != nil` (project lead: compile-time work happens once in Go, however cheap per call) | `vtRaw(...) catch \|err\| err: { log.warn; break :err true }` is: on a non-nil error, warn and `skip = true` |
| Labeled block that is a whole prong of the outer `switch (input.final)` / `switch (action.final)` and is left from inside a nested switch (`mode:`, `decrqm:`, `capture:`, `push:`, `set:`, `decsasd:`, `blk:`) | the outer Go switch carries the label `prong:` and `break :x` is `break prong` | equivalent because nothing follows the outer switch in csiDispatch/escDispatch. Each site has a `// break :x` comment |
| Value-producing inner block (`ansi:`, `question:` in `h`, `l`, `n`) | if / else if / else assigning the value; the failing branch does what Zig's does (`break prong` or `return nil`) | |
| `ansi:` of DECRQM (`p`), left from inside a nested switch | `goto ansi`, label after the block | the mapping's existing goto rule |
| `csi_param:` labeled `if` in nextNonUtf8, left with `break :csi_param` from the `else` of a switch | `goto csi_param` to a label after the `if` | same rule |
| `std.meta.intToEnum(csi.EraseDisplay, p) catch null` (exhaustive enum) | switch on the u16 value over the five enum values | the check is on the u16, before any conversion, so a value above 255 cannot be truncated into a valid one |
| `std.meta.intToEnum(csi.TabClear, p) catch` (non-exhaustive `enum(u8)`) | `if p > 0xFF { error branch }` | `std.enums.fromInt` of a non-exhaustive enum only checks that the value fits the tag type |
| `std.math.cast(u5, p) orelse` | `if p > 0x1F` | |
| `inline 22, 23 => \|number\|` | `case 22, 23:` with `number := input.Params[0]`; the `else => @compileError` arm is `panic("unreachable")` | |
| `@max(1, p)` | Go builtin `max(1, p)` on uint16 | |
| `std.unicode.utf8ValidateSlice` | `utf8.Valid` | same equivalence as `isSafeUtf8` (osc/encoding.zig section): both reject overlongs, surrogates and values above U+10FFFF |
| Exhaustive `switch` over an enum / union (`Parser.Action`, `osc.Command`, `csi.EraseDisplay`, `ansi.ProtectedMode`, `ansi.ModifyKeyFormat`, the action tag of `CSI = u`) | `default: panic("unreachable")` | as in osc.go |
| Non-exhaustive `switch` with `_ =>` (`csi.EraseLine`, `csi.TabClear`) | `default:` | |
| `{}` of a value of a non-exhaustive enum in a log line | `@enumFromInt(%d)` | Zig 0.15.2 `printEnumNonexhaustive` prints `.name` for a named value and `@enumFromInt(N)` otherwise; both `_` prongs are only reached by unnamed values |

#### Logging

- `log.warn` is `warn("stream", ...)`. `log.err` (the one in `CSI t` with no
  parameters) also goes through `warn("stream", ...)`, marked `// Zig: log.err`:
  the package has one hook, and an error-level message is printed whenever a
  warning is. The lead may prefer a separate hook.
- `log.info` (C1 execute, change icon, the `debug`-only lines) and
  `log.debug` (unimplemented OSC callbacks) are not printed; the calls are
  kept as comments, as in `search_screen.go` and `osc_parsers_iterm2.go`.
- Formats: `{f}` is `Format()` with `%s` (`ParserActionCSI`, `ParserActionESC`,
  `ParserAction`); `{s}` of the intermediates is `%s` of the `[]uint8`; `{}`
  of an integer is `%d`; `{any}` of the intermediates (configureCharset) is
  `zigFmtAnySlice`; `{}` of an error is `error.%v`; `0x{x}` is `0x%x`.

#### `nextSlice`, `nextSliceCapped` and the decode stand-in

- `nextSlice` is the scalar loop only: `comptime debug or !build_options.simd`
  is true for the reference build. `streamDebug` (Zig `debug`) is declared but
  only referred to in comments (its two uses are `log.info`).
- `nextSliceCapped` is ported line for line (the tests call it directly). Its
  call `simd.vt.utf8DecodeUntilControlSeq` is the file-private stand-in
  `streamUtf8DecodeUntilControlSeq` (result `streamDecodeResult{consumed,
  decoded}`, simd/vt.zig's `DecodeResult`). Upstream's scalar fallback
  `utf8DecodeUntilControlSeqScalar` is NOT ported, because it is the cause of
  the abort on `FF ESC [ 1 A` (stream.zig:593, upstream-findings section 12)
  and of the `E2 82 (` disagreement with the per-byte path; docs/mapping.md
  requires nextSliceCapped to return what nextSlice returns.
- The stand-in decodes the bytes before the first ESC with a fresh
  `UTF8Decoder`, feeding a byte again when the decoder does not consume it
  (as nextUtf8 does). `consumed` ends after the last complete or rejected
  sequence; a sequence still open at the end of the input is left for
  nextSliceCapped's `nextUtf8` loop (it stays pending in the stream's
  decoder); a sequence still open when the ESC is reached gets the U+FFFD the
  decoder emits on rejecting the ESC, and the bytes up to the ESC are
  consumed, so the ESC is handled by `consumeAllEscapes` as `handleCodepoint`
  would. This keeps nextSliceCapped's control flow and makes it identical to
  the per-byte path. `TestStream_go_only_nextSliceCapped_equals_per_byte_next`
  (`stream_capped_equiv_test.go`) checks it on 30,000 seeded inputs of
  0-23 bytes over an alphabet of ESC, CSI/OSC/DCS bytes, C0, invalid and
  truncated UTF-8 leads, continuation bytes, surrogate, overlong and
  above-U+10FFFF starts, each fed in random chunks (cp buffer of 4096 or
  exactly the chunk length): same codepoints, actions, parser state and
  decoder state as byte-by-byte `next`. A failure prints its seed;
  `TERM51_CAPPED_SEED=<n>` replays it. Dropping the U+FFFD for a sequence
  broken by ESC makes it fail at seed 37.

#### Other per-function notes

- `Execute`: `ansi.C0` is `enum(u7)` non-exhaustive; `ANSIC0(c)` after the
  `c > 0x7F` check. The C1 path builds `ParserActionESC{Intermediates:
  []uint8{}, ...}` (Zig `&.{}`).
- nextNonUtf8's CSI-param fast path: `*|=` / `+|=` are `saturatingMulU16` /
  `saturatingAddU16`; `params_sep.set(int(params_idx))`.
- `Print` takes `uint32` (u21); `handleCodepoint`'s `@intCast(c)` to u8 for
  `execute` is a plain conversion after `c <= 0xF`.
- Payload widths: the usize payloads (`print_repeat`, `delete_chars`, ...)
  are `uint(input.Params[0])`.
- `Stream.init` copies Zig's whole doc comment (the OSC 52 example and the
  size-limit sentence were missing from the wireframe).

## Small files and sets module: implementation notes

Recorded by the implementers of each file and folded in here (2026-10-05).

### Zig std hashing: `std.hash.Wyhash`, `std.hash.int`, `std.hash.autoHash`

Three new files port the parts of Zig 0.15.2's std (the toolchain that built
the recorded corpus) that `style.zig`, `hyperlink.zig` and `hash_map.zig`
reach. Nothing in them is ported from a Ghostty file. The Go standard
library has nothing equivalent (`hash/maphash` is a different, randomly
seeded function), and the hashes decide when a set or map reports that it
is full (`data/sets/style_hash.txt`, `hyperlink_hash.txt`), so they must be
bit-exact.

| Zig std file | Go file | Prefix |
|---|---|---|
| `lib/std/hash/wyhash.zig` | `terminal/std_hash_wyhash.go` | `stdHashWyhash` |
| `lib/std/hash.zig` (`int`) | `terminal/std_hash.go` | `stdHash` |
| `lib/std/hash/auto_hash.zig` | `terminal/std_hash_auto_hash.go` | `stdHashAutoHash` |

#### `std_hash_wyhash.go` (whole of `Wyhash`)

- Struct `Wyhash` is the type `stdHashWyhash` with Zig's field names. Methods
  keep Zig's names in lower case (the type is package-private, as in
  `std_fmt_parse_float.go`): `update`, `final`, `shallowCopy`, `smallKey`,
  `round`, `final0`, `final1`, `final2`. Functions without `self`:
  `stdHashWyhashInit(seed)`, `stdHashWyhashHash(seed, input)`,
  `stdHashWyhashRead(bytes, data)`, `stdHashWyhashMum(a, b *uint64)`,
  `stdHashWyhashMix`. The constant `secret` is `stdHashWyhashSecret`.
- `total_len` (usize, enters the result in `final2`) is `uint64` on every
  platform: the width of the 64-bit build that recorded the data. `buf_len`
  and indexes are `uint`.
- `mum`'s `@as(u128, a) *% b` is `bits.Mul64` (the full product; it cannot
  wrap). `std.mem.readInt(.little)` is `encoding/binary.LittleEndian`.
  `read`'s comptime `bytes` is an ordinary parameter (4 or 8 at every call;
  a general loop covers other widths). `input[i..][0..48]` (a `*const
  [48]u8`) is the Go conversion `(*[48]uint8)(input[i:][:48])`.
- `std.debug.assert` is `assert`. `final1`'s `input_lb.len - start_pos`
  would trap on underflow in Zig; the port asserts `start_pos <= len` before
  it. `undefined` fields are Go zero values (always written before read).
- `inline fn`s are ordinary Go functions.

#### `std_hash.go` (`std.hash.int`, u64 only)

- `int` is generic over its integer type and chooses the mixer by bit width
  at compile time. `Style.hash` uses it only on a u64, so the u64
  instantiation is written out: `stdHashIntU64(input uint64) uint64`, which
  calls the private mixer `uint64` (`stdHashUint64`). The unused `mult`
  constant, the `0...2` and `else` prongs and the `uint16`/`uint32` mixers
  are other instantiations and not ported. Go's `*` on uint64 is Zig's `*%`.
- `autoHash`, `autoHashStrat` and `Wyhash` are re-exports in hash.zig; their
  Go counterparts are in the other two files (`std.hash.autoHashStrat` is
  `auto_hash.hash`, i.e. a `stdHashAutoHashHash...` function).

#### `std_hash_auto_hash.go` (instantiations reached by `PageEntry.hash`)

- `hash`, `hashPointer`, `hashArray` and `autoHash` dispatch on the key type
  at compile time. Each instantiation the library reaches is written out as
  one Go function named after the Zig function plus the key type, keeping
  the branches that key type takes; the comptime decisions
  (`hasUniqueRepresentation`, `@typeInfo` prong, `typeContainsSlice`) are
  comments.

  | Zig call (in `hyperlink.PageEntry.hash`) | Go |
  |---|---|
  | `autoHash(&h, std.meta.activeTag(self.id))` (enum(u1)) | `stdHashAutoHashAutoHashEnumU1(&h, uint8(tag))` -> `stdHashAutoHashHashEnumU1` |
  | `autoHash(&h, v)` (v: `size.OffsetInt` = u32) | `stdHashAutoHashAutoHashU32(&h, v)` -> `stdHashAutoHashHashU32` |
  | `autoHashStrat(&h, slice, .Deep)` (`[]const u8`) | `stdHashAutoHashHashSliceU8(&h, slice, stdHashAutoHashHashStrategyDeep)` -> `stdHashAutoHashHashPointerSliceU8` -> `stdHashAutoHashHashArraySliceU8` -> `stdHashAutoHashHashU8` per byte, then `stdHashAutoHashHashUsize(len)` |

- `HashStrategy` is `stdHashAutoHashHashStrategy` (`...Shallow`, `...Deep`,
  `...DeepRecursive`) with a `String()` from a name table.
- Bytes hashed: the enum(u1) tag is 1 byte (enums have a unique
  representation, `@sizeOf` is 1); a u32 is its 4 little-endian bytes; a
  `[]const u8` with `.Deep` is every byte through a separate `update` call,
  then the length as a usize. **usize is hashed as 8 bytes on every
  platform** (the recorded data's width); Zig built for 32-bit ARM would
  hash 4 bytes and give different hyperlink hashes (checked: e.g.
  `autoHashStrat("hello", .Deep)` is 0x759b537a1dec968b on aarch64,
  0x2c43e210a5850fd1 on arm).
- `std.mem.asBytes(&key)` is native byte order in Zig; every port target
  (arm64, 32-bit ARM) and the recording build are little-endian, and the Go
  code writes little-endian explicitly.
- The hasher (`anytype`) is always `*std.hash.Wyhash` in the library, so the
  Go parameter is `*stdHashWyhash`.
- `hashPointer` of a slice with `.Shallow` hashes the slice's address
  (`@intFromPtr`), which Go cannot take without `unsafe` and the library
  never does: the Go function panics `not ported` there.
- `stdHashAutoHashHashEnumU1` asserts its argument is 0 or 1 (a u1 cannot
  hold more).
- `hash_map.AutoContext` does not reach this file: every key type it is used
  with (u32, i32, u64, usize, `Offset(Cell)` = packed struct(u32)) has a
  unique representation, so `std.hash_map.getAutoHashFn` returns
  `Wyhash.hash(0, std.mem.asBytes(&key))`, i.e. `stdHashWyhashHash(0, <key
  bytes LE at the Zig width>)` (usize 8 bytes).
- Not ported: every other instantiation (pointers, optionals, arrays,
  vectors, structs, unions, error unions), `typeContainsSlice`.

#### Tests

- `std_hash_wyhash_test.go`: all 5 tests of wyhash.zig ("test vectors",
  "test vectors at comptime", "smhasher", "iterative api", "iterative
  maintains last sixteen"). `verify.smhasher` and `verify.iterativeApi`
  (lib/std/hash/verify.zig) are ported into the test file for Wyhash. The
  comptime halves run a second time at run time.
- `std_hash_test.go`: `test int`, u64 line only (the other ten lines use
  other instantiations; they are kept as comments). The unnamed `test { _ =
  ...; }` block has no assertions and is not translated.
- `std_hash_auto_hash_test.go`: none of auto_hash.zig's 12 tests can be
  translated (they hash only key types that are not ported; the list with
  the type each needs is in the file header).
- Go-only tests (values printed by Zig 0.15.2, see "Cross-check"):
  `TestStdHashWyhash_go_only_seeded_inputs_match_zig`,
  `TestStdHash_go_only_int_u64_matches_zig`,
  `TestStdHashAutoHash_go_only_autoHash_u32`, `..._autoHash_enum_u1`,
  `..._enum_u1_out_of_range_panics`, `..._hash_usize_is_8_bytes`,
  `..._autoHashStrat_slice_u8`, `..._slice_shallow_panics`,
  `..._page_entry_sequence_matches_zig`.

#### Cross-check against Zig 0.15.2

A Zig program (term51 toolchain, built for aarch64 and
arm-linux-musleabihf, Debug and ReleaseSafe; run natively and under
qemu-arm) generated inputs with SplitMix64 and printed digests (every
result folded into one Wyhash, 8 LE bytes per result). The Go-only tests
recompute the same digests:

- Wyhash: 5000 inputs, random seed, length 0..300; for each, `Wyhash.hash`,
  one `update` + `final`, and `update` in random chunks of 0..63 bytes with a
  `final` after every chunk (all folded). Same digest on all four Zig builds
  (0x314f82606e8f4b36) and in Go native and on 32-bit ARM.
- `std.hash.int(u64)`: 15000 values (x, x & 0xffff, x << 40 for 5000 random
  x) plus 0, max and 0xdeadbeef. Same on all Zig builds and in Go.
- `PageEntry.hash` sequence (tag, implicit u32 or explicit id slice, uri
  slice; 5000 random entries, ids 0..120 bytes, uris 0..300 bytes) plus three
  fixed entries: Go equals 64-bit Zig (0x118773d096266e81). 32-bit Zig
  differs, as expected (usize length is 4 bytes there).

### `size.zig`, `point.zig`, `Tabstops.zig`

#### `size.zig` (`size.go`)

- Nothing to implement: the file keeps only the integer types and
  `max_page_size` (wireframe decision, unchanged). Header changed from
  "WIREFRAME" to "Port of". 2 of 24 Zig tests translated, 22 listed in
  `scripts/not-translated.txt` (Offset, OffsetBuf, getOffset, intFromBase).

#### `point.zig` (`point.go`)

- `Tag.String()` reads the name table `pointTagNames`, indexed by value, of
  length `pointTagCount` (= `@typeInfo(Tag).@"enum".fields.len`, the
  `iota` after the last tag), as `ParserState` does. A value outside the
  enum panics (index out of range), as `@tagName` of an invalid value does.
- `Point.coord` (`inline fn`, one prong for all four tags capturing the
  payload) is a switch on `Tag` returning the matching field; `default:
  panic("unreachable")` for a tag outside the enum.
- `Coordinate.eql` is `x == other.x && y == other.y`.

#### `Tabstops.zig` (`Tabstops.go`)

- `masks` (computed at compile time with `@shlExact(1, @as(u3, i))`) is
  the package-level `tabstopsMasks`, filled once at start-up by a function
  literal (`1 << i` for i in 0..7; nothing is shifted out, so `@shlExact`
  cannot trap).
- `entry`/`index` (`inline fn`): `col / 8`, `col % 8` (`@mod` on usize).
- `set`, `unset`, `get`: the `assert(dynamic_i < self.dynamic_stops.len)`
  are `assert`s, so a column beyond the dynamic storage stops the port
  exactly where Zig's safe build stops. `unset` keeps upstream's `^=` (it
  toggles; term51 tests pin the defect). `get`'s labeled block `unit:` is
  an if/else assigning `unit`.
- `resize`: the allocator is dropped and Go's `make` cannot fail, so it
  always returns nil; `make` zeroes (Zig's `@memset(new, 0)`).
  `fastmem.copy(Unit, new, old)` is `copy(new[0:len(old)], old)` (slicing
  traps like `@memcpy` would if `old` were longer). The free of the old
  storage is left to the GC. The quirks are kept: the dynamic storage is
  `cols - 512` units (bytes), not bits, and `size < len` is strict, so a
  resize to the current dynamic length reallocates. The local `new` keeps
  Zig's name (it shadows Go's builtin inside the function only).
  `resize_tw` and `tw.check(.dynamic_alloc)` are omitted (wireframe
  decision: test-only fault injection; the tripwire test and the three
  FailingAllocator tests are in `scripts/not-translated.txt`).
- `init`: on a `resize` error (unreachable in Go) returns the zero Tabstops
  and the error (Zig's value is undefined).
- `deinit`: the free is the GC's; `self.* = undefined` is `*self =
  Tabstops{}`.
- `reset`: `@memset(..., 0)` is Go's `clear`. `while (i < self.cols - 1) :
  (i += interval)`: `self.cols - 1` is a usize subtraction that traps in
  Zig when `cols == 0` (only reached with `interval > 0`), so the port
  panics ("integer overflow") before the loop; `i += interval` traps on
  overflow in Zig, so the port checks `i > maxUint - interval` before
  adding. `usize` here is Go `uint` (the platform width, as Zig's on each
  target; nothing in this file is recorded data).
- `Tabstops.resize`: Zig's local `new` is `new_` (Go builtin name); the
  `// Zig:` comments quote Zig's name.
- `TestTabstops_panic_reset_cols_zero` (Go-only) covers the `cols - 1`
  underflow in `reset` directly (`TabstopsInit(0, 8)`).

### `ref_counted_set.zig`

File: `terminal/ref_counted_set.go`.

| Zig | Go | Notes |
|---|---|---|
| `if (comptime @hasDecl(Context, "deleted")) ctx.deleted(v)` | `if self.has_deleted { any(ctx).(RefCountedSetContextDeleted[T]).Deleted(v) }` | **added field** `has_deleted bool`, decided once in `RefCountedSetInit` (same ruling as `vt_raw` in stream.go). The interface conversion at the call is what Go needs to reach a method outside the constraint; it runs only when the context has the hook (hyperlink set), never for the style set |
| `.{}` of an Item (`items[id] = .{}`, `item.* = .{}`, `@memset(items, .{})`) | `RefCountedSetItemInit[T, Id, Ref]()` | bucket = `^Id(0)` (maxInt) |
| `std.math.maxInt(Id)` | `^Id(0)` | |
| `@sizeOf(Id)` | `bits.Len64(uint64(^Id(0))) / 8` | 1, 2 or 4 |
| `std.math.ceilPowerOfTwoAssert(usize, cap)` | `uint(1) << bits.Len(cap-1)` after `assert(bits.Len(cap-1) < bits.UintSize)` | the assert can only fire on a 32-bit uint with a u32 Id |
| `std.math.log2_int(usize, x)` | `bits.Len(x) - 1` | |
| `std.mem.alignForward(usize, x, a)` | `(x + a - 1) &^ (a - 1)` after asserting `a` is a power of two | std asserts the alignment |
| `@intFromFloat(@ceil(@as(f64, n + 1) / load_factor))` | `uint(math.Ceil(float64(n+1) / RefCountedSetLoadFactor))` | float kept as in Zig: 0.8125 = 13/16 exactly, IEEE division is correctly rounded in both languages, so the integer results are identical (`data/sets/capacities.txt`, all 65,537 `c` and `l` lines, pass natively and on 32-bit ARM). Asserts for Zig's traps: `n + 1` overflow, result not fitting usize |
| `@intFromFloat(load_factor * @as(f64, table_cap))` | `uint(float64(RefCountedSetLoadFactor * float64(table_cap)))` | one product, explicit float64() against fusion (none possible: no add) |
| `@as(Id, @intFromFloat(@as(f64, cap) * 0.9))` (rehash threshold) | `Id(float64(float64(cap) * rehash_threshold))` with an assert that it fits Id | the untyped 0.9 rounds to the same f64 as Zig's comptime_float coerced to f64 |
| `(p +% 1) & mask`, `(hash +% i) & mask` | plain Go `+` on the same-width unsigned type | wrapping as in Zig |
| `for (0..self.max_psl + 1) \|i\|` | `for i := uint(0); i < uint(self.max_psl)+1; i++` | max_psl <= 31 (psl_stats bound), so the Id addition cannot overflow |
| `if (comptime std.debug.runtime_safety) assert(lookup == null)` in `insert` | `assert(!self.LookupContext(value, ctx).Valid)` | the reference is a safe build; costs one extra lookup per insert (see optimisation list) |
| `assertIntegrity` | empty body, `_ = ctx` | upstream's body is `if (false and ...)`, never compiled |
| `@branchHint(.cold)` / `.unlikely` | comment `// Zig: @branchHint(...)` | |
| `self.psl_stats.len - 1` | `len(self.psl_stats)-1` | |

Run-time checks added for Zig's overflow traps (each `assert` sits right
before the operation and is commented `// Zig: += traps on overflow`):
`meta.ref += 1` (add, addWithId, use), `meta.ref += n` (useMultiple),
`living -= 1` (release, releaseMultiple), `held_item.meta.psl += 1`
(insert). Not added where the value is bounded by an invariant of the set
itself: `next_id += 1` (`next_id < layout.cap`), `psl_stats[...] +=/-= 1`,
`items[...].meta.psl -= 1` (guarded by `> 0`), `max_psl -= 1` (guarded).
`releaseMultiple`: Zig's `item.meta.ref -= n` mixes `RefCountInt` and `Id`;
Go compares as uint64 and subtracts `Ref(n)` (n fits Ref: Zig rejects the
instantiation otherwise).

Zero capacity: `LookupContext` returns null when `layout.table_cap == 0`
(Go-only guard, comment in place). Zig reads `table[0]` from the memory
that follows (the test fixture keeps spare zero bytes). Every add on such a
set returns `ErrOutOfMemory` through upstream's own `next_id >= cap` check
(`living 0 < @intFromFloat(0 * 0.9)` is false), so no other guard is
needed; `insert` (whose `table_cap - 1` would underflow) is never reached.

Upstream quirk kept: `deleteItem` returns early only when
`bucket > table_cap` (not `>=`), so an item in the default state with
`bucket == maxInt(Id) == table_cap - 1` (u16 set laid out for 65,536) would
go on to `assert(table[bucket] == id)`.

Tests: ref_counted_set_test.go 79 + corpus_sets_test.go 1
(TestRefCountedSet_corpus_capacities), all pass natively and on 32-bit ARM.

### `style.zig`

File: `terminal/style.go`.

| Zig | Go | Notes |
|---|---|---|
| `Color.eql`: `@as(Tag, self) == @as(Tag, other) and switch (self) {...}` | `if self.Tag != other.Tag { return false }` then the switch | same short-circuit |
| `Color.format(self, fmt, options, writer)` (old 4-argument signature, so Zig 0.15 never calls it through `{f}`/`{any}`) | `Format() string` | `{}` of a u8 is decimal (`itoa`) |
| `Style.format`: `inline for (std.meta.fields(Style))` / `(std.meta.fields(Flags))` | the fields written out in declaration order: a loop over the three colours, then the `flags` block (eight bools in order, then `underline` and `_padding`, which are not bool) | `!std.meta.eql(value, d_val)` on a Color is `Color.Eql` (tag, then active payload: what std.meta.eql does for a tagged union); for the enum and the u5 it is `!=` |
| `{any}` of a Color (`printValue`, Zig 0.15.2, depth 3) | local closure `fmtAny` in `Style.Format` | `.{ .palette = 5 }`, `.{ .rgb = .{ .r = 1, .g = 2, .b = 3 } }`, `.{ .none = void }` (never printed: none is the default) |
| `{any}` of `sgr.Attribute.Underline` | `"." + underline.String()` | `printEnumExhaustive`; an out-of-range Go value panics in String() (Zig: unreachable) |
| `{any}` of the u5 `_padding` | `itoa` | |
| `VTFormatter.format(self, writer) !void` / `HtmlFormatter.format` | `Format() string` (wireframe) whose body is the Zig body as a local `format := func(writer *oscWriter) error`, run on an `oscWriterAllocating`; a non-nil error is `panic("unreachable")` (an allocating writer never fails) | keeps every `try` as `if err := ...; err != nil { return err }` |
| `writer.print("\x1b[{d};5;{d}m", .{...})` etc. in the two `formatColor` | one `writer.writeAll` of the same bytes built with `itoa` (Zig's print line in a comment above) | std's print formats each integer into a local buffer and writes it with writeAll, so the byte stream is the same; with a fixed writer that runs out, `fixedDrain` fills the buffer to the brim either way, so even the partial result is identical |
| `switch (bold) { .bright => {}, .color => \|v\| break :default v.toTerminalRGB() }` (labeled blocks `default:`, `palette:`, `rgb:` in `fg`) | `return` from the Go switch arm, `// break :default` | each labeled block is a whole prong of the `return switch` |
| `@intFromEnum(color.Name.bright_black)` | `uint8(ColorNameBrightBlack)` in a local const | |
| `PackedStyle.Data.fromColor`: `inline else => \|v, t\| @unionInit(Data, @tagName(t), ...)` | switch on the tag returning the 24-bit value of that view | none 0; palette idx in bits 0..7; rgb r, g<<8, b<<16 |
| `@bitCast(packed_style)` to `[2]u64` | `stylePackedStyle.bitCast` (wireframe stub) | the fields shifted into place; `underline` masked to 3 bits, `_padding` to 5 and each Data to 24, so the Go values that the Zig widths cannot hold are dropped as the packed layout would (they cannot exist in Zig) |
| `bgCell` `.content = .{ .color_palette = idx }` | `PageCellContent{color_palette: idx}` | the other views zero (Zig: the rest of the 24 bits of the packed union are not set) |
| `Style.hash`: `@call(.always_inline, std.hash.int, .{wide[0] ^ wide[1]})` | `stdHashIntU64(wide[0] ^ wide[1])` (std_hash.go) | `data/sets/style_hash.txt`: all 4,096 styles match, natively and on 32-bit ARM (TestStyle_corpus_hash) |

`StyleColorTag.String` uses a names array (`styleColorTagNames`), as sgr.go.

### `hyperlink.zig`

File: `terminal/hyperlink.go`.

| Zig | Go | Notes |
|---|---|---|
| `Hyperlink.deinit(alloc)` | `Deinit()`: the switch kept, nothing freed | stub kept because the tests call it (wireframe) |
| `alloc.dupe(u8, s)` | `make([]uint8, len(s))` + `copy` | never fails; a duplicated empty slice is empty and non-nil |
| `errdefer` after a successful step | the cleanup written before the later error return, `// Zig: errdefer ...` (the `osc_parsers_color.go` idiom) | `PageEntry.dupe`: an allocation failure of the explicit id frees the uri copy. The errdefers with nothing failing after them are comments only |
| `.{ .offset = size.getOffset(u8, dst_page.memory, &buf[0]), .len = uri.len }` | `buf[0:len(uri)]` | the allocation from its first byte; its capacity reaches the end of the last chunk (BitmapAllocator requirement), which nothing here uses |
| `x.slice(base)` / `offset.ptr(base)[0..len]` | `x` itself | Go-memory rule |
| `self.page.?` / `(self.src_page orelse self.page.?)` in the Set context | `assert(self.page != nil)` (for `hash`, only when `src_page` is nil) | Zig's "attempt to use null value" panic; Go-only panic tests `TestHyperlink_panic_SetContext_*` |
| `std.meta.activeTag(x)` | `x.Tag` | |
| `std.mem.eql(u8, a, b)` | `bytes.Equal(a, b)` | |
| `PageEntry.hash`: `Wyhash.init(0)`, `autoHash(&h, activeTag(id))`, `autoHash(&h, v)` (implicit, u32), `autoHashStrat(&h, slice, .Deep)` (explicit id, then uri), `final()` | `stdHashWyhashInit(0)`, `stdHashAutoHashAutoHashEnumU1(&h, uint8(tag))` (the tag is an enum(u1): explicit 0, implicit 1), `stdHashAutoHashAutoHashU32`, `stdHashAutoHashHashSliceU8(&h, s, stdHashAutoHashHashStrategyDeep)`, `h.final()` | `data/sets/hyperlink_hash.txt` has no Go test (the Zig dump test is not translated, and no corpus test reads the file); checked with a throw-away test (not committed): all 3,000 lines match natively and on 32-bit ARM. Suggest adding a corpus test like TestStyle_corpus_hash |

Tag `String()` methods use names arrays (`hyperlinkHyperlinkIdTagNames`,
`hyperlinkPageEntryIdTagNames`).
- `data/sets/hyperlink_hash.txt` is checked by the Go-only
  `TestHyperlink_corpus_hash` (`corpus_hyperlink_test.go`), which builds the
  3,000 entries directly (the Go entry holds its slices, no page needed); the
  Zig dump test that produced the file is not translated.

## Page layer: implementation notes

Recorded by the implementers of each file and folded in here (2026-10-05).

### `bitmap_allocator.zig`

File: `terminal/bitmap_allocator.go`.

| Zig | Go | Notes |
|---|---|---|
| `comptime assert(std.math.isPowerOfTwo(chunk_size))` | `assert(chunk_size != 0 && chunk_size&(chunk_size-1) == 0)` in `BitmapAllocatorInit` | run time in Go (Go-only panic test) |
| `@sizeOf(T)`, `@alignOf(T)` (comptime) | `reflect.TypeFor[T]().Size()` / `.Align()`, read ONCE in `BitmapAllocatorInit` into the **added fields** `elem_size`, `elem_align` (the `has_deleted` precedent) | `BitmapAllocatorBytesRequired` has no receiver and reads `@sizeOf(T)` per call (`bitmapAllocatorSizeOf`). Go sizes equal Zig's for every element type used (u8, u16, u21 = uint32, u64, [3]u8). Go's alignment differs only for 8-byte scalars on 32-bit ARM (Go 4, Zig 8), read only by alloc's `chunk_size % @alignOf(T)` assertion with a chunk size below 8; the page uses 16 and 32 |
| `@intFromPtr(slice.ptr) - @intFromPtr(chunks)` (free, isAllocated) | `reflect.ValueOf(slice).Pointer()` minus the same for `chunks` (`bitmapAllocatorAddr`), in the helper `chunkIndexOf` (not a Zig declaration; the shared expression of free and isAllocated) | addresses are only subtracted, never turned back into pointers; no package `unsafe`. Zig's traps kept: `-` underflow (slice before `chunks`, or a nil slice) and `@divExact` (not on a chunk boundary). Go-only guard: a start past the end of `chunks` panics (Zig would write bitmap words through the unchecked `[*]u64`) |
| `bitmaps[i] \|= ~@as(u64, 0) >> @intCast(64 - bits) << @intCast(bit)` with `bits == 0` (empty slice) | `assert(bits > 0)` before the shift | Zig's `@intCast(64)` to u6 traps; Go would shift by 64 and continue (Go-only panic test) |
| `for (1..n)` in findFreeChunks with `n == 0` | `assert(n >= 1)` inside the per-bitmap loop, where Zig's `for` is | so an empty bitmap list still returns null for n = 0, as in Zig |
| findFreeChunks, large path: the read of `bitmaps[i]` after the middle loop | `if i >= len(bitmaps) { return None }` | the term51 reference fix (`patches-reference/0001`); the pinned upstream aborts (out-of-bounds read at bitmap_allocator.zig:281). Tests `TestBitmapAllocator_findFreeChunks_large_request_past_last_bitmap` and `..._random_bitmaps_skipped_inputs_no_space` |
| `continue :search` from the inner `while (rem > 64) : (i += 1)` | labeled `continue search` | Go's labeled continue skips the inner post statement, as Zig's does |
| `std.math.mul(usize, @sizeOf(T), n) catch return error.OutOfMemory` | `bits.Mul64`, high word non-zero is `ErrOutOfMemory` | 64-bit as the reference |
| `std.math.divCeil(usize, byte_count, chunk_size)` | `(byte_count-1)/chunk_size + 1` for a positive numerator, else `byte_count/chunk_size` | std's unsigned branch; its error (zero denominator) cannot happen |
| `findFreeChunks(..., chunk_count)` with `chunk_count` a 64-bit usize | Go-only: a count above `math.MaxUint` returns `ErrOutOfMemory` before the call | only on a 32-bit target; such a count exceeds every chunk the bitmaps hold, so findFreeChunks would return null |
| `ptr[0..n]` at `&chunks[idx * chunk_size]` | `chunks[start : start+n : end]`, `start`/`end` the byte offsets of the first chunk and of the end of the last chunk divided by `@sizeOf(T)` | capacity reaches the end of the last chunk (wireframe REQUIREMENT). Go-only assert: the chunk starts on a whole element (an element size that does not divide the chunk size cannot be a Go `[]T`; never with the page's types) |
| `std.mem.alignForward(usize, x, a)` | `bitmapAllocatorAlignForward` (not a Zig declaration) | asserts `a` is a power of two (std) and that `x + (a - 1)` does not overflow (Zig's `+` traps) |
| `a * b` on usize in layout/bytesRequired | `bitmapAllocatorMul` (not a Zig declaration) | asserts no overflow (Zig's `*` traps) |
| `@divExact` whose exactness follows from an alignForward just before | plain `/`, comment `// @divExact: exact by construction` | |
| `@popCount`, `@clz`, `@ctz` | `bits.OnesCount64`, `bits.LeadingZeros64`, `bits.TrailingZeros64` | |
| `std.log.warn("bm={b} idx={}", ...)` (dumpBitmaps) | `warn("default", "bm=%b idx=%d", ...)` | unscoped std.log is the `default` scope |
| `comptime assert(@import("builtin").is_test)` (isAllocated) | not checked | the function is unexported and only tests call it |
| `init`: `.chunks = buf.member(u8, l.chunks_start)` | `chunks = make([]T, bitmap_count * 64 * chunk_size / @sizeOf(T))` | what the bitmaps can hand out (file header) |

Tests: bitmap_allocator_test.go 74 + corpus_bitmap_test.go 3, all pass
natively and on 32-bit ARM (qemu).

### `hash_map.zig`

File: `terminal/hash_map.go`. The wireframe's structure (metadata slice +
shared `hdr` pointer, value receivers, `hashMapEntrySizes`, `keyIndex`) is
kept unchanged; no signature changed.

| Zig | Go | Notes |
|---|---|---|
| `AutoContext.hash` = `getAutoHashFn(K)`: `Wyhash.hash(0, std.mem.asBytes(&key))` (every key type used has a unique representation) | type switch on `any(key)` writing the key little-endian at its Zig width into an 8-byte stack buffer, then `stdHashWyhashHash(0, buf[:n])` | u8/i8 1, u16/i16 2, u32/i32 4, u64/i64 8 bytes; Go `uint`/`int` (usize/isize) always 8 (int sign-extended). Any other key type panics `not ported`. The switch is a per-call run-time step standing for Zig's comptime width: Go cannot specialise a method of the empty generic struct `hashMapAutoContext[K]` per type argument, and the struct has no field to hold a choice made once. A defined type (e.g. `type X uint32`) would hit the default; the library's keys are `SizeOffsetInt` (alias of uint32) |
| `AutoContext.eql` = `std.meta.eql(a, b)` | `a == b` | integers |
| `Metadata` packed struct, `@as(u8, @bitCast(self))` in isFree / isTombstone | method `bitCast()` (not a Zig declaration): `fingerprint&0x7f \| (used&1)<<7` | the masks drop what a u7 / u1 cannot hold |
| `takeFingerprint`: `@truncate(hash >> (64 - 7))` | `uint8(hash >> 57)` | |
| `self.metadata.?` (null unwrap, a safety panic) | `assert(self.metadata != nil)` in `header()` and `initMetadatas()`; `removeByIndex` and the probe loops index the slice (a nil slice panics on the index) | Go-only panic tests `TestHashMap_panic_*_null_metadata` |
| `self.metadata.? + idx` then `metadata[0]` (unchecked many-pointer) | `self.metadata[idx]` | idx is masked below capacity = len(metadata) |
| `const mask = self.capacity() - 1` (u32) | `assert(capacity >= 1)` before the subtraction | in putAssumeCapacityNoClobberContext, getIndex, getOrPutAssumeCapacityAdapted (capacity-0 panic tests) |
| `self.capacity() - self.header().size` in growIfNeeded | `assert(capacity >= size)` | Go-only test: a full map overfilled by an assume-capacity put |
| `self.count() + additional_size` in ensureUnusedCapacity | `assert(additional_size <= MaxUint32 - count)` | |
| `header().size += 1` / `-= 1` (u32) | assert before (`< MaxUint32` / `> 0`) | removeByIndex panic test |
| `x.* = undefined` (fetchRemoveAdapted, removeByIndex, getOrPutAssumeCapacityAdapted) | Go zero value of K / V | mapping table: `undefined` is the zero value |
| `@memset(metadata[0 .. capacity], 0)` (initMetadatas) | `clear(self.metadata[0:self.Capacity()])` | |
| Iterator `while (metadata != end) : ({ metadata += 1; it.index += 1; })` | `for ; len(metadata) != 0; metadata, it.index = metadata[1:], it.index+1` over `metadata[it.index:cap]` | the return inside the body skips the post statement as in Zig |
| FieldIterator `metadata += 1; items += 1` | reslice `[1:]` of both | |
| keyIterator / valueIterator with null metadata: `metadata` and `items` `undefined` | nil slices, `len: 0` | |
| `init`: metadata at `buf + @sizeOf(Header)`, keys/values at `layout.keys_start` / `vals_start` | `make` of each, `layout.capacity` long; header allocated and pointed at by `hdr` | capacity 0 gives a non-nil empty metadata slice (wireframe REQUIREMENT) |
| `OffsetHashMap.init`: `getOffset(..., m.metadata.?)`; `map(base)`: `self.metadata.ptr(base)` | copy `metadata` and `hdr` from / to the unmanaged map | |
| `removeByPtr`: `if (@sizeOf(K) > 0) (@intFromPtr(key_ptr) - @intFromPtr(self.keys())) / @sizeOf(K) else 0` | `keyIndex` holds the whole expression, with package `unsafe` as the wireframe allows (`unsafe.Sizeof(*key_ptr)`, `unsafe.Pointer`, `unsafe.SliceData`) | panics when the pointer is before `keys` (Zig's `-` trap) or the element at the computed index is not `key_ptr` (wireframe REQUIREMENT; Go-only foreign-pointer test) |
| comptime `key_align` / `val_align` (`1` for a zero-sized type), `header_align` (4), `base_align` (max) | computed at the top of `hashMapLayoutForCapacity` from `hashMapEntrySizes` | the function is the only user; a page computes it once per layout |
| `std.mem.alignForward(usize, x, a)` in layoutForCapacity | local closure `alignForward`, asserting a power-of-two alignment (std) | no overflow possible: cap <= 2^31 |
| `assert(new_capacity == 0 or std.math.isPowerOfTwo(new_capacity))` | `assert(new_capacity == 0 \|\| new_capacity&(new_capacity-1) == 0)` | Go-only panic tests (layoutForCapacity, OffsetHashMap.layout) |
| The Context-less wrappers (`put`, `get`, ... : compile error unless `@sizeOf(Context) == 0`, then `undefined` context) | `var ctx C` (zero value) | |
| `getOrPutAdapted`'s compile-time check | comment only | |

Tests: hash_map_test.go 101 + hash_map_2_test.go 51 +
corpus_hashmap_test.go 1 (data/hashmap/hashmap_data.txt), all pass natively
and on 32-bit ARM (qemu).

### `page.zig`

File: `terminal/page.go`.

#### Memory

| Zig | Go | Notes |
|---|---|---|
| `Page.memory`, `Offset(Row)`, `Offset(Cell)` | `rows []PageRow`, `cells []PageCell` (one slice, `capacity.rows * capacity.cols`) | wireframe decision; `Row.cells` keeps the byte offset `cells_start + index*8` |
| `row.cells.ptr(self.memory)` (many-item pointer to the row's first cell) | `self.rowCells(row)`: `self.cells[(row.cells - cells_start)/8:]`, sliced by the caller as Zig slices the pointer | private helper, not a Zig declaration. A row of another page whose offset is out of range panics when sliced |
| `cells_start` outside `layout` | `pageCellsStart(cap)`: `alignForward(cap.rows * @sizeOf(Row), @alignOf(Cell))`, the first lines of `layout` | computing the whole layout per call would be slow (it runs the set layouts' float formulas) |
| `getOffset(Cell, self.memory, cell)` | `self.cellOffset(cell)` = `pageCellsStart + cellIndex(cell)*8`; asserts the result fits u32 (`@intCast` to OffsetInt) | |
| pointer subtraction (`cellIndex`, `rowIndex`) | `unsafe` pointer arithmetic, the one sanctioned use (wireframe): panics if the pointer is not an element of the page's slice (also for an empty slice) | Go-only panic tests `GetCells` of another page's row rely on it |
| `initBuf(buf, l)` | private `pageInitBuf(rows, cells, l)`: the buffer is the rows and cells slices; the sub-structures make their own memory | the wireframe omitted initBuf; it is reinstated (private) because `init` and `reinit` both call it and `reinit` must reuse the page's own rows and cells: Zig zeroes the same memory, so a `*Row`/`*Cell` held elsewhere (for example Screen's cursor during `PageList.eraseRows`' reinit of the last page) still points into the page. A fresh allocation would leave such pointers on a detached copy. `PageList`'s prune path (`first.data = .initBuf(.init(buf), layout)`) can use it the same way |
| `init`: `posix.mmap` + `errdefer munmap` | `make` of rows and cells; the error result is always nil | `assert(l.total_size % page_size_min == 0)` kept |
| `deinit`: `munmap`, `self.* = undefined` | `*self = Page{}` | as `Tabstops.deinit` |
| `reinit`: `@memset` of the memory as u64, then `initBuf` | `clear(rows)`, `clear(cells)`, `pageInitBuf(self.rows, self.cells, layout(capacity))` | |
| `clone` / `cloneBuf` (mmap + memcpy of the whole page) | `Clone`: copy of the struct, then of every slice the page owns: rows, cells, both allocators (bitmaps and chunks), both hash maps (metadata, a new header, keys, values) and both sets (table, items). Private helpers `pageCloneBitmapAllocator`, `pageCloneHashMap`, `pageCloneRefCountedSet` | the copies reach into the private fields of bitmap_allocator.go, hash_map.go and ref_counted_set.go (same package) |
| an `Offset(T).Slice` copied with the memory (grapheme map values, hyperlink entries' uri and explicit id) | `pageRebase(s, from, to, chunk_len)`: finds `&from[i] == &s[0]` among the chunk starts of the old allocator and returns `to[i : i+len(s) : i+cap(s)]` | no `unsafe` (not sanctioned here), so a linear scan of chunk starts per slice (graphemes: up to 512 starts for 8192 bytes). Capacity kept (appendGrapheme grows into it). An empty slice, or one not inside the allocator (not page memory), is returned unchanged |
| `fastmem.copy(Cell, dst, src)` | `copy(dst, src)` | Go's copy is a memmove; upstream's memcpy with overlapping ranges is undefined (it is never reached with overlap in the library's own calls as far as checked) |
| `@memset(@as([]u64, @ptrCast(cells)), 0)` | `clear(cells)` | the zero PageCell is the all-zero cell |

#### Byte arithmetic (`layout`, `Capacity`)

- All byte sizes and offsets are `uint64` (wireframe). The sizes and base
  alignments Zig reads from types are untyped constants next to the
  wireframe's `...Sizes` values: `pageSizeOfRow`/`pageSizeOfCell`/
  `pageAlignOfCell` 8, `pageBitSizeOfRow`/`pageBitSizeOfCell` 64,
  `pagePageSizeMin` 4096, and the base alignments:
  StyleSet 8 (RefCountedSet's `@max` includes `@alignOf(Layout)`, which has
  usize fields), GraphemeAlloc and StringAlloc 8 (`@alignOf(u64)`),
  GraphemeMap 8 (`@max(Header 4, key 4, value 8)`), hyperlink.Set 8,
  hyperlink.Map 4 (`@max(4, 4, 2)`). The wireframes of the three
  sub-structure files omit `base_align`; page.go needs the numbers for the
  `alignForward`/`alignBackward` calls.
- `std.mem.alignForward/alignBackward(usize, ...)` are `pageAlignForward` /
  `pageAlignBackward` (Zig 0.15.2: assert the alignment is a power of two;
  alignForward's `addr + (alignment - 1)` asserted not to overflow).
  `std.math.divCeil(usize, ...)` is `pageStdMathDivCeil` (unsigned branch;
  every caller's `catch` is for a zero divisor, which never happens).
  `std.math.ceilPowerOfTwo(T, v) catch unreachable` and
  `ceilPowerOfTwoAssert(u32, v)` are `pageStdMathCeilPowerOfTwo(v, bits)`
  (asserts `v != 0` and that the result fits T). They could move to zig.go
  and be shared with ref_counted_set.go's inline versions.
- `layout`: `hyperlink_count` is a u16 (`@divFloor` of the u16 field);
  `hyperlink_count * hyperlink_cell_multiplier` is a u16 product of at most
  1365 * 16, so it cannot trap and `std.math.cast(u32, ...) orelse
  maxInt(u32)` never takes its `orelse` (comment in place).
  `@intCast(grapheme_count)` to the map's u32 Size is asserted.
- `availableBitsForGrid`: the usize subtractions cannot underflow (each
  backward start is at or after the forward start of the same member); no
  assert added, comment in place.
- `adjust`: `@intCast(new_rows)` to u16 is asserted
  (`TestPg1_panic_Capacity_adjust_new_row_count_overflows_u16`).
- `exactRowCapacity`: the four `@intCast`s of the result are asserted
  (`..._styles_capacity_overflows_u16`). `std.StaticBitSet(65536)` is the
  private `pageCellCountSet` (1024 u64 masks, `set`, `isSet`, `count`; a
  u16 index is always in range).

#### Packed structs `Row` and `Cell`

- `PageRow.bitCast` / `pageRowFromBits` and `PageCell.bitCast` /
  `pageCellFromBits` place every field at its Zig bit offset; a field wider
  in Go than in Zig is masked to the Zig width (u2 enums, the u21
  codepoint, the u23 `_padding`), as in style.go.
- `Cell.content` is an untagged packed union of three views of 24 bits; Go
  has one field per view. `pageCellFromBits` fills all three from the same
  bits. `bitCast` takes the view `content_tag` selects for its own bits
  (codepoint 0..20, palette 0..7, rgb 0..23) and the bits above that view
  from `color_rgb`, the one view covering all 24. This reproduces Zig for
  every cell built by a whole literal (other views zero) or read from raw
  bits, including the leftover bits 21..23 under a codepoint tag
  (`term51: Cell.isZero: leftover bits are not ignored`), and for a Go
  write of `content.codepoint` alone (Zig's field write keeps bits 21..23).
  It differs from Zig only for a Go value whose views were set
  inconsistently by hand (no library code does that).
- `Cell.init`: `@bitCast(@as(u64, 0))` is the Go zero value; the u21
  parameter is asserted `<= 0x1FFFFF`.
- Enum `String()`s read names arrays (`pageRowSemanticPromptNames`,
  `pageCellContentTagNames`, `pageCellWideNames`,
  `pageCellSemanticContentNames`); exhaustive switches on these enums
  (`hasText`, `codepoint`, `gridWidth`, `isEmpty`, the `wide` switch of
  `verifyIntegrity`) end in `panic("unreachable")`.

#### Control flow

- `clonePartialRowFrom`: the labeled blocks `hyperlink:` and `dst_id:` are
  left from inside nested blocks, so they are `goto hyperlink` / `goto
  dst_id` with the label after the block (mapping's goto rule); `style:` is
  the last statement of the loop body, so `break :style` is `continue`.
  `copy:` is a plain block assigning `*dst_row`. The deferred
  `pauseIntegrityChecks(false); assertIntegrity()` is a deferred closure;
  Go also runs it while a panic unwinds (Zig aborts first), which only
  matters to the panic tests and does not change that they panic.
- `addWithId... catch |e| switch (e) { ... } orelse id`: an error switch
  over `ErrOutOfMemory` / `ErrNeedsRehash` with `default:
  panic("unreachable")`, then `Valid ? Value : id`. The same shape for
  `string_alloc.alloc ... catch |err| switch (err)` in insertHyperlink.
  Where Zig only asserts the error set at compile time (`comptime
  assert(@TypeOf(e) == error{OutOfMemory})`), any error maps to the page
  error.
- `errdefer` (insertHyperlink, setGraphemes, appendGrapheme): written out
  before the later error returns, in Zig's unwinding order (the explicit
  id, then the uri); errdefers with nothing failing after them are
  comments. As upstream, a failed `addWithIdContext` in
  clonePartialRowFrom leaks the duplicated strings in the page's string
  allocator (no errdefer upstream).
- `moveCells`' parameter `len` hides Go's builtin inside the function;
  `src_cells.len` is written as `len` (the slice was cut to that length).
- `pauseIntegrityChecks`: `+= 1` / `-= 1` on the usize counter are guarded
  by asserts (Zig traps on overflow/underflow;
  `TestPg1_panic_Page_pauseIntegrityChecks_unpause_below_zero`).
- `assertIntegrity`: `log.err` goes through `warn("page", ...)` marked `//
  Zig: log.err` (the stream.go precedent), then `panic("page integrity
  violation")`. The DebugAllocator is dropped.
- `verifyIntegrity`: the Valgrind early return is dropped (comment). The
  arena and the two `std.AutoHashMap`s are Go maps; `getOrPut` + `+= 1` is
  `m[k] += 1`. Go's map order is not Zig's, so when several styles (or
  hyperlinks) have a short ref count, a different one may be logged first;
  the returned error is the same. The zombie-style block is kept with its
  unused counter (`_ = zombies`).
- `self.page.?`-style unwraps (`lookupGrapheme(...).?`, `getEntry(...).?`,
  `getPtr(...).?`) are `assert(x.Valid)` / `assert(p != nil)`.
- `build_options.kitty_graphics` blocks: not ported (wireframe), a comment
  marks each place.

#### Integrity checks are always on

`pageSlowRuntimeSafety` is the constant `true` (wireframe), so every
`assertIntegrity` (clearCells, clearGrapheme, clearHyperlink, setGraphemes,
appendGrapheme, setHyperlink, moveCells, swapCells, cloneFrom,
clonePartialRowFrom) runs a full `verifyIntegrity`, which hashes every cell
of the page into the hyperlink map. The recorded corpus comes from a build
where this is off. See the optimisation list.

#### Tests

- page_1/2/3_test.go: 405 Go tests (140 + 135 + 130): the 320 translated
  Zig tests (all 331 Zig tests are mapped; 11 listed in
  not-translated.txt) and the Go-only panic tests. hyperlink_test.go: 42
  tests (the 32 that needed PageInit included). All 447 pass, each run on
  its own, natively (arm64) and under 32-bit ARM qemu (main at 869098c
  merged). No translated test turned out to be layout-only beyond the 11
  already listed.
- `data/layout/page_layout.txt` (27,060 lines: 505 capacities with every
  Layout field and maxCols, and 3,325 `Capacity.adjust` results) matched
  line for line natively and on 32-bit ARM, checked with a throw-away test
  (not committed). Suggest a Go-only corpus test like
  TestStyle_corpus_hash; nothing reads the file today.

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
  The same limitation applies to `vtRaw` (project lead review, 2026-10-05):
  `vt_raw` is detected on the handler value given to `StreamInit`. For a
  value-type handler Go would call `VtRaw` on that copy, and a pointer-receiver
  `VtRaw` on a value handler is not detected at all, where Zig's
  `@hasDecl` would see it. Every handler in the package is a pointer, so
  nothing differs today.
- The reference-counted set is generic over all four of Zig's parameters, has
  no memory base, keeps `cap`, `table_cap` and `table_mask` as counts (the
  byte offsets are dropped), treats a zero-capacity set as empty (lookups find
  nothing, adds return `ErrOutOfMemory`), and keeps upstream's float
  formulas (`capacityForCount`'s `ceil((n+1)/0.8125)`, the `0.8125 *
  table_cap` items cap, the `cap * 0.9` rehash threshold), each product
  explicitly rounded so Go cannot fuse it (project lead, 2026-10-05: the
  faithful choice, replacing the earlier "integer arithmetic" wording). They
  must reproduce `data/sets/capacities.txt` (every capacity 0 to 65,536), and
  do, natively and on 32-bit ARM. `scripts/check-no-fma.sh` covers them: it
  checks the test binary too, because generic set code such as
  `RefCountedSetLayoutInit` is only instantiated by the tests until the page
  layer uses it.
- Hashes (`Style.hash`, `PageEntry.hash`) must reproduce
  `data/sets/style_hash.txt` and `data/sets/hyperlink_hash.txt`: they decide
  when a set reports that it is full and so drive page growth. `Style.hash`
  needs Zig's `std.hash.int`; `PageEntry.hash` needs Wyhash with `autoHash`
  semantics, neither of which the Go standard library has.

## Points are compared by tag and active coordinate (project lead, 2026-10-05)

`point.Point` is a Zig `union(Tag)`; the Go `Point` is a struct with all four
payload fields (`Active`, `Viewport`, `Screen`, `History`), of which only the
one named by `Tag` is meaningful. Go's `==` would also compare the three
inactive fields, so points are never compared with `==`: code compares
`Tag` and `Coord()` (`PointCoordinate.Eql`), and tests use
`expectEqualPoint` / `expectEqualOptionalPoint`
(`terminal/zig_expect_point_test.go`), which are Zig's `testing.expectEqual`
on a union: the tags, then only the active payload. On 2026-10-05 a
type-checked scan of the package and its tests found no `==` or `!=` on a
`Point` (or a struct or array containing one) and no `Point` map key in
non-test code; the 294 test assertions that used the generic `expectEqual`
on `Point` (286) or `Optional[Point]` (8) were switched to the two helpers.

`Style` flags follow the same idea for widths: `Style.Eql` (and so
`Default`) compares the flags as the masked u16 that `Style.hash` uses
(`styleFlags.bits`), and `Style.Format` prints those bits, so a Go value with bits above `underline`'s u3 or
`_padding`'s u5 compares, hashes and prints as its Zig bits
(`TestStyle_go_only_flags_out_of_range_bits_are_masked`).

## The one per-call exception to the compile-time rule (project lead, 2026-10-05)

`hashMapAutoContext.Hash` picks the key's Zig width with a type switch on
every call. Zig resolves `getAutoHashFn(K)` at compile time, but Go cannot
resolve it once for an empty generic context struct (no field to hold the
choice, no specialisation by type parameter). It is a type switch, not
string work. A defined key type (for example `type X uint32`) panics "not
ported"; the library's keys are aliases and are fine. Every other piece of
Zig compile-time work is done once in Go.

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
