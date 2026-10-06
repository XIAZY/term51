# Zig-shaped code to revisit after the port

The port follows the Zig source function by function, on purpose: that is
what lets every Go function be checked against upstream and every recorded
case match. The price is that a lot of the package reads like Zig written in
Go. This file lists those places, so they can be revisited deliberately once
the port is complete and verified.

Rules for revisiting, set by the project owner ("faithful port first, revisit
and optimise later"):

- Nothing here changes until the final gate has passed.
- Each item becomes its own change, with the full test suite and the
  recorded corpus (natively, on 32-bit ARM, and on the phone) proving that
  behaviour did not change.
- Behaviour changes, such as fixing upstream defects, are a separate list:
  `term51/docs/deliberate-fixes.md`. Performance items are in
  `post-port-optimizations.md`. This file is about shape: what a Go
  programmer would find foreign.

A recommended order is at the end.

## The target: faithful in behaviour, Go in form (project owner, 2026-10-06)

"A faithful port that makes sense in Go": the same behaviour as upstream,
proven by every test and recorded case, in code a Go programmer recognises
as Go. Measured that way, everything below falls into two groups.

**Change (shape only; behaviour identical):**

- Data types: `Optional[T]` becomes `(value, ok)` or pointers; the
  every-variant-in-one-struct unions become small types or interfaces; Go
  field names and receivers (sections 2.1 to 2.3).
- Control flow: `goto` for labelled blocks and the `errdefer` closures become
  ordinary Go structure (2.10, 2.11).
- Errors instead of panics wherever a caller's input can reach them; internal
  invariants stay as asserts, possibly debug-only (2.4).
- Zig leftovers removed: allocator parameters, `Deinit` methods that free
  nothing, errors that cannot happen (2.5, 2.6).
- Output through `io.Writer` / `bytes.Buffer` instead of the emulated Zig
  writer (3.1). Condition: where output depends on it, upstream's
  fixed-buffer behaviour stays (write what fits, then fail), as pinned by the
  partial-write tests.
- Configuration instead of package-level switches (5.1).
- Package layout and API: internal packages, a small exported `vt` API,
  everything else unexported (1.1 to 1.4).
- Hot paths written the Go way (`post-port-optimizations.md`).
- Go's standard library wherever it behaves identically to Zig's (3.2 to
  3.6).

**Keep (behaviour depends on it):**

- Zig's float parser and maths, its hashes and the UTF-8 decoder, where
  results are visible (3.2, 3.3, 3.5).
- The 64-bit integer rules on 32-bit ARM (2.9).
- The byte bookkeeping that decides scrollback retention (4.1); it may be
  rewritten in plain Go as long as it computes the same numbers.
- Upstream's quirks (`term51/docs/deliberate-fixes.md` decides those
  separately).

**The cost.** Reshaped code no longer lines up function by function with
upstream, so porting future Ghostty changes takes more work. To keep that
feasible, `mapping.md` keeps its Zig-to-Go correspondence up to date during
the pass (which Go function now carries each Zig function). The tests and
the recorded corpus remain the proof that behaviour is unchanged.

**How.** After the final sign-off, one "Go-native pass", module by module in
the order of the port. After each module the full test suite and every
recorded case must still match upstream, natively, on 32-bit ARM and on the
phone.

## 1. The public surface

1.1 **There is no idiomatic Go API yet.** Every exported name is a Zig
declaration with a Go prefix: `TerminalInit`, `Terminal.VtStream`,
`StreamInitAlloc[*StreamReadonlyHandler]`, `InputKeyEncodeEncode(writer,
event, opts)`. Callers must build Zig-style option structs, often with
non-zero Zig defaults that a Go literal does not give (see the "Defaults a
Go literal does not give" sections in the file headers).
*Revisit:* the thin API agreed on day one, a `vt` package with `New`,
`Write`, `Resize`, `Replies`, `Events`, a render-state snapshot, and key,
mouse, paste and focus encoders taking value types. The `terminal` package
then becomes an internal package.

1.2 **One very large package.** 84 source files (about 58,000 lines) and
140 test files (about 6,300 test functions) in `package terminal`, because Zig's file-level namespaces became name prefixes
instead of packages (`InputKeyModsRemapSet`, `OSCParsersKittyTextSizingOSC`,
`SearchSlidingWindow`). Some names are mechanically long, for example
`UucodeXTypesGraphemeBreakNoControl` and `InputKeyKeyKeyA`.
*Revisit:* split into internal packages along the module lines (parser,
page, screen, terminal, input, search, unicode, std stand-ins), which lets
the prefixes go.

1.3 **Exported names that should not be exported.** Exporting follows Zig's
`pub`, not what a Go caller needs. Examples: page and page-list internals
(`PageRow`, `PageCell`, `PageListNode`, `PageListPin` with all its methods),
`RefCountedSet`, the hash map types, and the debug-only `PinIsValid`.

1.4 **File names.** `PageList.go`, `Screen.go`, `Terminal.go` and others keep
Zig's file-struct capitalisation; Go convention is lower case.

## 2. Language idioms

2.1 **`Optional[T]` instead of Go idioms.** Zig's `?T` is a struct with
`Valid` and `Value` throughout (`Optional[ColorRGB]`,
`Optional[PageListPin]`, `Optional[[]uint8]`): 316 type uses, 338
`Some`/`None` calls and 446 `.Valid` checks in the non-test code. Go would use `(T, bool)`
returns, pointers, or a zero value with documented meaning.

2.2 **Tagged unions as structs with a `Tag` field and one field per
payload** (23 union types). For example `StreamAction`, `OSCCommand`, `SGRAttribute`, `Point`,
`PageListScroll` and `Selection` bounds. Every variant's payload field exists
in every value, which makes them large (this is also performance item 1),
and `==` compares inactive fields, which is why points must be compared
with `Eql`. Go would use interfaces with one type per variant, or smaller
dedicated types.

2.3 **Zig field names and receivers.** Struct fields keep Zig's `snake_case`
(`page_pin`, `pending_wrap`, `max_scrollback`), receivers are named `self`,
and an `X` prefix marks names that start with a digit (`X8Bg`,
`x132_column`).

2.4 **Panics for Zig's traps and assertions.** About 1,100 `assert` and
`panic` sites (601 `assert` calls and 261 `panic("unreachable")` among them) reproduce Zig's safety checks: integer overflow,
out-of-range casts, `unreachable`. A Go library normally returns errors and
does not panic on caller input. The upstream inputs that abort are already
ruled to return errors (`term51/docs/upstream-crashes.md`); the remaining
asserts guard internal invariants.
*Revisit:* decide per site whether it is an internal invariant (keep, maybe
behind a debug build tag) or reachable from a caller (return an error).

2.5 **Errors that cannot happen.** Functions keep Zig's error unions where
Go cannot fail: allocation errors with no allocator (27 `return ...
ErrOutOfMemory` sites), and `TabstopsInit` or `Resize` returning `error`.
There are 33 sentinel errors. Error values are bare sentinels named after
Zig's errors (`ErrOutOfMemory`, `ErrWriteFailed`) and are never wrapped,
which the replayer relies on.

2.6 **Allocator leftovers and `Init`/`Deinit` pairs** (42 `...Init`
constructors, 37 `Deinit` methods). `alloc any` parameters and fields exist only
to tell "has an allocator" from "has none" (`StreamInitAlloc`,
`OSCParserInit`), and `Deinit` methods exist that free nothing, kept because
tests call them.

2.7 **Explicit generic instantiation everywhere** (26 generic types). For example
`RefCountedSetInit[Style, StyleId, SizeCellCountInt, StyleSetContext](...)`
and `BitmapAllocatorInit[uint32](4, ...)`, mirroring Zig's compile-time type
parameters. Most would disappear if these became ordinary types for their one
or two uses.

2.8 **Value-type contexts and interfaces mimicking `@hasDecl`.**
`StreamHandlerVtRaw` and `RefCountedSetContextDeleted` are optional
interfaces checked once at init to stand for Zig's "does the type declare
this function". This works, but would be designed differently in Go.

2.9 **Fixed 64-bit widths where Go would use `int`.** Byte sizes and offsets
are `uint64`, scroll deltas are `int64`, and `usize` hashing is 8 bytes, so
the 32-bit phone matches the 64-bit reference. That is correct and should
stay; but the mix of `uint`, `uint64`, `SizeCellCountInt` (`uint16`) and
`int64` in signatures needs conversions at almost every call site.

2.10 **`goto` and closures for Zig's labelled blocks.** `break :label
value` and labelled `continue` have no Go equivalent, so the port uses 105
`goto` statements (21 labels such as `prong:`, `outer:`, `shiftLeft:`) and
closures returned from in place. Go would restructure these into helper
functions or early returns; the literal form exists so each function still
reads side by side with Zig.

2.11 **`errdefer` as deferred closures on named error results.** Zig's
`errdefer` (cleanup only when the function fails) becomes `defer func() {
if err != nil {...} }()` with a named `err` result (45 deferred closures, in
Zig's unwinding order), and block-scoped `defer` becomes a closure called in
place. Idiomatic Go would undo state explicitly on each error path or
restructure so there is nothing to undo.

2.12 **`// Zig:` comments.** 1,668 comments record the Zig construct each
Go line stands for. They are what makes review possible and should stay as
long as the code stays literal; an idiomatic rewrite of a function would
drop them.

## 3. Zig standard library stand-ins

Ported because Go's standard library behaves differently and the port has to
match Zig byte for byte. Each would be revisited as "keep, because behaviour
depends on it" or "replace with Go's".

3.1 **`oscWriter` / `oscWriterAllocating`** stand in for `std.Io.Writer`.
Writers in Go would be `io.Writer` or `*bytes.Buffer`. The fixed-buffer
partial-write semantics, write what fits and then fail, must be kept where
output depends on them.

3.2 **Number parsing**: `zigParseInt` and its per-file copies,
`std_fmt_parse_float.go` (all of Zig's float parser, including its `_`
defect). Required for exactness today. If the defect is fixed later
(deliberate fix 7), `strconv` could replace most of it.

3.3 **Hashing**: `std_hash_wyhash.go`, `std_hash.go`, `std_hash_auto_hash.go`.
Must stay as long as hash placement or `bindingHash` values are observable.

3.4 **Data structures with Go equivalents**: `hash_map.go` (Zig's offset
hash map with tombstones and fingerprints), `bitmap_allocator.go` over a
slice, `ref_counted_set.go`, `datastruct_circ_buf.go`, `StaticBitSet24/256`,
the insertion-ordered pin set and cell set, and the hand-written
insertion sort in `RemapSet.finalize`. Several exist only because upstream's
byte bookkeeping (option A) needs their exact fill points; the rest could be
Go maps, slices and `sort`.

3.5 **Zig's float maths.** `color.go` hand-ports Zig's `pow`, `cbrt`,
`exp` and `log` (Go's `math` gives different last bits), and every stored
product is explicitly rounded (`float64(x*y)`) so Go cannot fuse
multiply-adds, which Zig never does (`scripts/check-no-fma.sh` enforces it).
Go would use `math.Pow` and plain expressions. Keep while the recorded
colour data must match bit for bit.

3.6 **Small std helpers**: `std_mem.go`, `std_ascii.go`, `std_unicode.go`, and
the duplicated `splitScalar` and `tokenize` helpers per OSC sub-parser
(performance item 3).

## 4. Memory and pointers

4.1 **Byte bookkeeping on Go slices (option A).** Page layouts, set and map
layouts, and `page_size` are computed as upstream's byte figures, so
scrollback retention matches Ghostty exactly. That is a decision, not
an accident, but it is the most Zig-shaped part of the package.

4.2 **Pointer arithmetic stand-ins.** `cellIndex` and `rowIndex` use
`unsafe` (the one sanctioned use; 11 `unsafe.` references) and
`BitmapAllocator.Free` uses `reflect` address differences (12 `reflect.`
references in all, including `SetPwd`'s and the copy helpers' aliasing checks) to turn a pointer back into a position, as Zig
does with pointer subtraction. Keeping indexes instead of pointers in the
cursor and pins would remove both (performance items 14 and 24).

4.3 **Page memory reuse.** Zig's pools reuse node addresses and Go's garbage
collector does not, a documented, rare difference
(`mapping.md`, Page layer).

## 5. Global mutable state

5.1 **Package-level switches that tests flip:** `PageStdCapacity` (tests set
Zig's smaller test value), `pageSlowRuntimeSafety` (off by default, on in
`TestMain`), `platformGOOS` (tests switch platforms), and `logWarn`. Because
of them, tests cannot run in parallel, and two terminals in one program share
these settings.
*Revisit:* move them into configuration carried by the terminal or the
encoder options.

5.2 **Run-time stand-ins for compile-time work.** Besides the tables below:
the `vt_raw` and `has_deleted` fields (Zig's `@hasDecl`), `elem_size` and
`elem_align` in `BitmapAllocator` (Zig's `@sizeOf`/`@alignOf`), and the one
per-call type switch in `hashMapAutoContext.Hash`.

5.3 **Start-up tables** computed by package-level variable initialisers:
mode lookup, Unicode tables, the X11 color map, key tables, kitty entries.
This is fine in Go, but it is several megabytes initialised at program start
whether used or not.

## 6. Tests and tooling

6.1 **Tests are Zig-shaped.** Test names carry Zig titles and file numbering
(`TestKe3_...`, `TestPl2_...`), and helpers mirror Zig's `std.testing`.
They are the specification and must stay one to one with the Zig tests
while the port is being verified. After that, the Go test layout can be
reorganised as long as the mapping script keeps working.

6.2 **The corpus replayer** depends on internal field names and on error
names matching Zig's. Any renaming in sections 1 to 3 has to update the
replayer's dump code at the same time.

## 7. Platform

7.1 **Run-time platform choice.** The macOS and iOS keyboard behaviour is
chosen at run time from `platformGOOS`. That is correct for a
platform-neutral library. A Go-native alternative would be build tags, at
the cost of no longer testing all three modes in one binary.

7.2 **Memory limit on the phone.** On the BlackBerry phone the test process
ran out of memory without `GOMEMLIMIT` (about 2 GB of RAM; one test allocates
a 205 MB page). The term51 app must set a memory limit at start-up. This is a
requirement for the app, not a library change.

## Recommended order

1. **The idiomatic `vt` API (1.1)**, because it is what users see and it
   lets everything else become internal without breaking anyone.
2. **The performance list** (`post-port-optimizations.md`, items 1, 2 and 19
   first), measured against the baseline benchmark (stream: about 6 MB/s on
   this machine).
3. **Global state into configuration (5.1)**, which also allows parallel
   tests.
4. **Package split and export clean-up (1.2 to 1.4).**
5. **Idioms (section 2) and std stand-ins (section 3)**, file by file, each
   only where behaviour does not depend on the Zig version.

## Counts

The figures above are for the non-test code in `terminal/` (84 files, about
58,000 lines) at Go commit b40fcb9, 2026-10-06, counted with `grep`; they are
a size guide, not exact inventories.
