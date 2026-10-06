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

1.2 **One very large package.** About 100 files and about 5,000 tests in
`package terminal`, because Zig's file-level namespaces became name prefixes
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
`Optional[PageListPin]`, `Optional[[]uint8]`). Go would use `(T, bool)`
returns, pointers, or a zero value with documented meaning.

2.2 **Tagged unions as structs with a `Tag` field and one field per
payload.** For example `StreamAction`, `OSCCommand`, `SGRAttribute`, `Point`,
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
`panic` sites reproduce Zig's safety checks: integer overflow,
out-of-range casts, `unreachable`. A Go library normally returns errors and
does not panic on caller input. The upstream inputs that abort are already
ruled to return errors (`term51/docs/upstream-crashes.md`); the remaining
asserts guard internal invariants.
*Revisit:* decide per site whether it is an internal invariant (keep, maybe
behind a debug build tag) or reachable from a caller (return an error).

2.5 **Errors that cannot happen.** Functions keep Zig's error unions where
Go cannot fail: allocation errors with no allocator, and `TabstopsInit` or
`Resize` returning `error`. Error values are bare sentinels named after
Zig's errors (`ErrOutOfMemory`, `ErrWriteFailed`) and are never wrapped,
which the replayer relies on.

2.6 **Allocator leftovers.** `alloc any` parameters and fields exist only
to tell "has an allocator" from "has none" (`StreamInitAlloc`,
`OSCParserInit`), and `Deinit` methods exist that free nothing, kept because
tests call them.

2.7 **Explicit generic instantiation everywhere.** For example
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

3.5 **Small std helpers**: `std_mem.go`, `std_ascii.go`, `std_unicode.go`, and
the duplicated `splitScalar` and `tokenize` helpers per OSC sub-parser
(performance item 3).

## 4. Memory and pointers

4.1 **Byte bookkeeping on Go slices (option A).** Page layouts, set and map
layouts, and `page_size` are computed as upstream's byte figures, so
scrollback retention matches Ghostty exactly. That is a decision, not
an accident, but it is the most Zig-shaped part of the package.

4.2 **Pointer arithmetic stand-ins.** `cellIndex` and `rowIndex` use
`unsafe` (the one sanctioned use) and `BitmapAllocator.Free` uses
`reflect` address differences to turn a pointer back into a position, as Zig
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

5.2 **Start-up tables** computed by package-level variable initialisers:
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
