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
- `osc_parser` is an `osc.Parser`. Until `osc.zig` is ported, `osc.go` is a
  marked placeholder that recognises nothing (see `docs/port-status.md`).
