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
- `osc_parser` is an `osc.Parser`. Until `osc.zig` is ported, `osc.go` is a
  marked placeholder that recognises nothing (see `docs/port-status.md`).

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
  implementation is approved those tests fail by design; run the ones that can
  pass with `go test ./terminal/ -skip 'SGR|Color|X11|KittyColor|MouseShape'`.
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
