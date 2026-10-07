# Optimisations deferred until the port passes

The port is literal on purpose, and some of the literal translations are
slower in Go than the original is in Zig. Nothing here may be changed until
every translated test and every recorded case passes; then each item is its
own change, measured before and after.

## Baseline (before any change)

`terminal/bench_test.go` (`go test ./terminal -run '^$' -bench . -benchtime 2s`),
at the signed-off port plus the benchmark commit (d7720fb), 2026-10-06. The
stream benchmarks run the parser and stream with a handler that does
nothing; the Terminal benchmarks feed the same bytes to a real 80x24
terminal with the page self-checks off, as a release build has them. Inputs
of about 128 KB: plain ASCII lines, coloured text (SGR), and a mixed stream
with cursor movement, erases and CJK wide characters.

| Benchmark | Development machine (arm64, 12 cores) | BlackBerry phone (32-bit ARM, 4 cores) | allocs/op |
|---|---:|---:|---:|
| Stream, ASCII | 16.7 MB/s | 5.46 MB/s | 0 |
| Stream, SGR | 6.2 MB/s | 1.93 MB/s | 0 |
| Stream, mixed | 3.2 MB/s | 1.48 MB/s | 0 |
| Terminal, ASCII | 5.3 MB/s | 1.45 MB/s | 54 |
| Terminal, SGR | 3.9 MB/s | 1.02 MB/s | 37 |
| Terminal, mixed | 2.5 MB/s | 0.83 MB/s | 21 |

The phone ran with `GOMEMLIMIT=900MiB`. Every change below is measured
against these numbers, on both machines.

## Changes made

How the numbers are taken. The development machine has slow and fast cores,
so an unpinned run varies by a factor of two (the table above was unpinned).
From here on the native numbers are the best of 5 runs of one second each,
pinned to one of the two fastest cores (`taskset -c 1`, `-test.cpu 1`). The
phone numbers are the best of 3 runs of one second. On the phone the stream
numbers repeat within 1%; the Terminal numbers vary by about 10% from run to
run, so a Terminal difference smaller than that is not a result.

Native, MB/s:

| Benchmark | Baseline (d7720fb) | Items 1, 2 (4682b6e) | Item 31 | Item 32 |
|---|---:|---:|---:|---:|
| Stream, ASCII | 25.1 | 25.0 | 40.3 | same |
| Stream, SGR | 6.5 | 19.3 | 24.0 | same |
| Stream, mixed | 4.8 | 23.1 | 26.2 | same |
| Terminal, ASCII | 6.1 | 6.0 | 6.1 | 11.6 |
| Terminal, SGR | 4.0 | 6.9 | 7.1 | 9.9 |
| Terminal, mixed | 3.1 | 6.3 | 6.4 | 7.8 |

Phone, MB/s:

| Benchmark | Baseline (d7720fb) | Items 1, 2 (4682b6e) | Item 31 | Item 32 |
|---|---:|---:|---:|---:|
| Stream, ASCII | 5.46 | 5.61 | 13.48 | same |
| Stream, SGR | 1.93 | 5.84 | 7.66 | same |
| Stream, mixed | 1.48 | 6.28 | 8.13 | same |
| Terminal, ASCII | 1.45 | 1.83 | 1.80 | 3.10 |
| Terminal, SGR | 1.02 | 1.83 | 1.69 | 2.17 |
| Terminal, mixed | 0.83 | 1.59 | 1.52 | 1.83 |

Allocations per run did not change (0 for the stream; 54, 37 and 21 for the
Terminal).

- Items 1 and 2 (4682b6e): `Parser.nextInto` writes the three action slots
  into storage the stream owns instead of returning them by value, and the
  deferred closure is gone. Sequences are three to five times faster; plain
  text, which never reaches the parser, is unchanged.
- Item 31: the handler receives a pointer to one `StreamAction` the stream
  owns and reuses, instead of a 624-byte value per action. The stream is up
  to 2.4 times faster on the phone. The Terminal numbers do not move beyond
  the phone's run-to-run variation: with the stream out of the way, a
  Terminal run is now almost entirely `Terminal.Print` and the functions it
  calls (about 80% of the native profile on plain text, of which the `defer`
  in `Terminal.Print` alone is about 12%). That is the next item.
- Item 32: the integrity check that `Terminal.Print`, `Terminal.printCell`
  and `Screen.CursorRight` deferred is now an ordinary call at the end of
  each. Plain text through a terminal is 1.7 times faster on the phone and
  1.9 times natively, more than the `defer`'s share of the profile; where
  the rest comes from has not been broken down. A Zig `defer` does
  not run when the function panics, so the ordinary call is also the closer
  translation. The stream benchmarks do not reach this code.

## Against upstream (2026-10-07)

Upstream's own library (v1.3.1, 332b2ae, unmodified), built with Zig 0.15.2
on the development machine and fed the same three inputs through the same
two paths, pinned to the same core, best of 5 runs of one second. The
benchmark is `bench/` in the term51 repository. Upstream "plain" is
`-Dsimd=false`, the code path the port follows; both upstream columns are
ReleaseFast. MB/s:

| Benchmark | Upstream SIMD | Upstream plain | Go baseline (d7720fb) | Go at item 32 | Upstream plain / Go |
|---|---:|---:|---:|---:|---:|
| Stream, ASCII | 187.9 | 65.9 | 25.1 | 40.3 | 1.6 |
| Stream, SGR | 71.3 | 52.8 | 6.5 | 24.0 | 2.2 |
| Stream, mixed | 74.1 | 51.4 | 4.8 | 26.2 | 2.0 |
| Terminal, ASCII | 37.2 | 28.3 | 6.1 | 11.6 | 2.4 |
| Terminal, SGR | 30.7 | 26.1 | 4.0 | 9.9 | 2.6 |
| Terminal, mixed | 27.9 | 23.7 | 3.1 | 7.8 | 3.0 |

Upstream plain built ReleaseSafe (safety checks on, the build the recorded
cases came from): Stream 65.9 / 51.6 / 50.5, Terminal 27.1 / 24.5 / 22.0.

Through a terminal, upstream's SIMD path gains 1.2 to 1.3 times over its
plain path, so the remaining gap is in the per-character screen work, not
in scanning bytes. The goal for the next speed step (project lead,
2026-10-07) is within 1.5 times of upstream plain through a terminal.
There is no phone column: upstream's Zig is not built for the phone.

## Items

| # | Where | What is slow | Why it is that way now |
|---|---|---|---|
| 1 | `Parser.Next` and `Stream` | Returns `[3]Optional[ParserAction]` per input byte. `ParserAction` is a struct holding every payload, including a whole OSC command by value, so three of them are copied for every byte, and again on the way through `Stream`. Measured by the project lead on 2026-10-05: parser and stream run at about 6 MB/s on ordinary coloured output (zero allocations); these copies account for most of the time in the profile (upstream: hundreds of MB/s). First item of the optimisation phase. | Zig returns `[3]?Action`, a small tagged union; the struct-with-tag mapping makes it large. |
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
| 14 | `BitmapAllocator.Free` / `isAllocated` | `reflect.ValueOf(slice)` per call (may move the slice header to the heap) to take the address difference Zig gets from `@intFromPtr`. | The file header does not sanction `unsafe`; `reflect.Value.Pointer()` gives the address without it. |
| 15 | `hashMapAutoContext.Hash` | A type switch on the key per call to pick the Zig width, then the general Wyhash path. A 4-byte specialisation for the page's u32 keys would be faster. | Go cannot decide the width once for an empty generic struct. |
| 16 | `hash_map` `getOrPutAssumeCapacityAdapted`, `getIndex` | `Capacity()` (nil check + header read) inside the probe loop. | Zig does the same; read it once. |
| 17 | `BitmapAllocator.findFreeChunks` small-request path | Shifts the bitmap n-1 times per bitmap; doubling shifts would take about log n steps. | Literal translation. |
| 18 | `hash_map` `initMetadatas` / Init | Keys and values are separate allocations; one combined allocation would reduce GC load. | Ordinary Go memory instead of one page buffer. |
| 19 | `pageSlowRuntimeSafety` | When on, every clearCells, setHyperlink, setGraphemes, moveCells, ... runs a full `verifyIntegrity` that hashes every cell of the page. | Resolved by the project lead ruling of 2026-10-05: a package variable, false by default as in upstream's release builds (the recorded corpus came from ReleaseSafe, where it is off); the test binary switches it on in `TestMain`, so tests and replay keep the long checked runs (docs/mapping.md "Integrity checks follow upstream's release build"). |
| 20 | `PageList` `pageListStdSize` / `pageListMemoryLen` | Recompute `PageLayoutOf` (with the set's float formulas) on every page create, destroy, grow prune, compact and `minMaxSize`. Cache per capacity, or store the size on the node. | Literal translation; Zig computes `std_size` at compile time and reads the memory length off the page. |
| 21 | `PageList` pin set | Linear search for `PutNoClobber` and `SwapRemove`; Zig's is a hash map. Only matters with many tracked pins. | Insertion-ordered slice keeps Zig's iteration order simply. |
| 22 | `PageList.verifyIntegrity` and `Grow` | `PinIsValid` for every tracked pin walks the page list (pins x pages per check), and `Grow`'s fast path runs a full page integrity check per row; with `pageSlowRuntimeSafety` on, these dominate long scroll tests (slowest native test 15 s; ARM PageList_2 9 minutes). | Upstream design; only with `pageSlowRuntimeSafety` on, which production leaves off and the test binary turns on (item 19). |
| 23 | `pageRebase` (page Clone) | Scans chunk starts to re-point slices into the copied allocators; an allocator index helper would make it O(1). | No `unsafe` in page.go beyond `cellIndex`/`rowIndex`. |
| 24 | `Screen` cursor | Steps the cursor by resolving `cellIndex` on each move; keeping the cell index instead of a pointer avoids the lookup. | Zig steps a pointer; the port keeps pointer-shaped code with index resolution. |
| 25 | `Selection.order`, `topLeft`, `bottomRight` | `contains` / `containedRow` do six or more page-list walks per call (`pointFromPin` twice per `order`). | Literal translation; upstream's own NOTE says the same. |
| 26 | `formatter.go` pin/point maps | With a map set, each format runs twice (the discarding pass only counts bytes); the counts could be computed directly. | Literal translation of the two-pass writer. |
| 27 | `formatter.go` per-cell output | `writeCodepoint` and `formatStyleOpen` build strings per call (an allocation per cell); `formatterAppendNTimes` appends in a loop; palette output is 256 `writeAll` calls. | Each Zig `print` is one `writeAll` of the same bytes. |
| 28 | `render.go` cell set | Membership by linear scan, so `LinkCells` is quadratic in linked cells; an index map beside the slice fixes it. | Insertion-ordered slice keeps Zig's iteration order simply. |
| 29 | `render.go` `UpdateHighlightsFlattened` | rows x highlights x chunks loop. | Upstream flags it as slow too. |
| 30 | `Terminal.printAttributes` / `input_key_mods` fixed buffer streams | Two private fixed-buffer-stream stand-ins could be one shared helper. | Ported in parallel branches. |
| 31 | `StreamHandler.Vt` | Every action, every printed character included, built a `StreamAction` (a struct with a field per payload, 624 bytes) and passed it to the handler by value. Done: the handler now gets a pointer to an action the stream owns and reuses, valid only during the call (see "Changes made"). | Zig passes a small tagged union; the struct-with-tag mapping makes it large. |
| 32 | `Terminal.Print`, `Terminal.printCell`, `Screen.CursorRight` | A `defer` of the screen integrity check runs for every printed character. Done for these three (see "Changes made"); the other functions that defer the check (cursor moves, erases, scrolls) run once per sequence, not per character, and keep the `defer`. | Zig's `defer` costs nothing; Go's costs a deferred call and blocks inlining. |
