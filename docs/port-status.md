# Port status

One row per Zig file. A file is "ported" when its Go file exists, every
upstream test and every project test for it is translated and passes, and the
recorded cases that exercise it pass, on this machine and as a 32-bit ARM
build.

| Zig file | Go file | Code | Zig tests translated | Corpus | 32-bit ARM |
|---|---|---|---|---|---|
| `terminal/parse_table.zig` | `terminal/parse_table.go` | done | 20 of 20, all pass | table equals upstream's (3,584 entries) | pass |
| `terminal/UTF8Decoder.zig` | `terminal/UTF8Decoder.go` | done | 15 of 15, all pass | 69,207 of 69,207 decode cases | pass |
| `terminal/Parser.zig` | `terminal/Parser.go` | done | 73 of 73, all pass | 9,010 of 9,010 parser traces (`TestCorpusReplaySelfParserTrace`) | pass |
| `terminal/csi.zig` | `terminal/csi.go` | done | no tests | - | pass |
| `terminal/dcs.zig` | `terminal/dcs.go` | done | 30 of 30; 29 pass, 1 skips as upstream does (tmux compiled out) | - | pass |
| `terminal/apc.zig` | `terminal/apc.go` | done | 12 of 12; 8 pass, 4 skip as upstream does (Kitty graphics compiled out) | - | pass |
| `terminal/osc.zig` | `terminal/osc.go` | done | 32 of 36 (4 not translated), all pass | OSC dispatch in the parser traces above | pass |
| `terminal/osc/encoding.zig` | `terminal/osc_encoding.go` | done | 6 of 6, all pass | `isSafeUtf8` equal to Zig on all 3- and 4-byte strings | pass |
| `terminal/osc/parsers.zig` | `terminal/osc_parsers.go` | done (declarations only) | no tests | - | pass |
| `terminal/osc/parsers/*.zig` (15 files) | `terminal/osc_parsers_*.go` | done | 448 of 457 (9 not translated), all pass except 1 that skips as upstream does (iterm2) | - | pass |
| `os/string_encoding.zig` | `terminal/os_string_encoding.go` | done | 50 of 50 (18 upstream, 32 term51), all pass | - | pass |
| `terminal/sgr.zig` | `terminal/sgr.go` | done | 60 of 64 (4 not translated), all pass | - | pass |
| `terminal/color.zig` | `terminal/color.go` | done | 68 of 71 (3 not translated), all pass | `data/colors` (lab grid, lerp, 256-colour generation) bit for bit; default palette table | pass |
| `terminal/x11_color.zig` | `terminal/x11_color.go` | done | 8 of 8, all pass | `rgb.txt` byte for byte | pass |
| `terminal/kitty/color.zig` | `terminal/kitty_color.go` | done | 9 of 10 (1 not translated), all pass | - | pass |
| `terminal/mouse_shape.zig` | `terminal/mouse_shape.go` | done | 7 of 7; 6 pass, 1 skips as upstream does (library build) | - | pass |
| `terminal/ansi.zig` | `terminal/ansi.go` | done | 9 of 9, all pass | - | pass |
| `terminal/charsets.zig` | `terminal/charsets.go` | done | 9 of 9, all pass | - | pass |
| `terminal/modes.zig` | `terminal/modes.go` | done | 23 of 25 (2 not translated: a layout test and an analysis-only unnamed test), all pass | modes table equals upstream's | pass |
| `terminal/device_status.zig` | `terminal/device_status.go` | done | 7 of 7, all pass | - | pass |
| `terminal/kitty/key.zig` | `terminal/kitty_key.go` | done | 25 of 25, all pass | - | pass |
| `terminal/stream.zig` | `terminal/stream.go` | done | 253 of 264 (11 not translated: the C-binding `cval` tests and `test Action`), all pass | needs Terminal (replay areas) | pass |
| `terminal/size.zig` | `terminal/size.go` | done | 2 of 24 (22 not translated, listed), all pass | - | pass |
| `terminal/point.zig` | `terminal/point.go` | done | 6 of 6, all pass | - | pass |
| `terminal/Tabstops.zig` | `terminal/Tabstops.go` | done | 35 of 39 (4 allocation-failure tests not translated), all pass; 1 Go-only panic test | - | pass |
| `terminal/ref_counted_set.zig` | `terminal/ref_counted_set.go` | done | 79 of 80 (1 not translated), all pass | capacities.txt: all 65,537 capacities match | pass |
| `terminal/style.zig` | `terminal/style.go` | done | 106 of 108 (2 not translated), all pass | style_hash.txt: 4,096 of 4,096 | pass |
| `terminal/hyperlink.zig` | `terminal/hyperlink.go` | done | 35 of 37 (2 not translated), all pass (42 tests with the Go-only ones) | hyperlink_hash.txt: 3,000 of 3,000 | pass |
| `terminal/bitmap_allocator.zig` | `terminal/bitmap_allocator.go` | done | 74 translated and Go-only, all pass | corpus_bitmap_test.go: 3 pass | pass |
| `terminal/hash_map.zig` | `terminal/hash_map.go` | done | 152 translated and Go-only, all pass | corpus_hashmap_test.go passes | pass |
| `terminal/page.zig` | `terminal/page.go` | done | 320 of 331 (11 not translated), all pass, plus Go-only panic tests (405 page tests in all) | page_layout.txt: 27,060 of 27,060 lines | pass |
| std `hash/wyhash.zig`, `hash.zig` (`int`), `hash/auto_hash.zig` | `terminal/std_hash_wyhash.go`, `std_hash.go`, `std_hash_auto_hash.go` | done (int: u64; autoHash: the cases PageEntry.hash reaches) | wyhash 5 of 5, int 1 (u64 lines), auto_hash 0 of 12 (key types not ported); 9 Go-only tests pinning Zig output | - | pass |
| std `fmt/parse_float.zig` + `parse_float/*.zig` (f64) | `terminal/std_fmt_parse_float.go` | done | 6 of 12 std tests (6 f16/f32/f80/f128-only not translated); 2 Go-only; bit for bit with Zig on 11,643,449 inputs | osc/rgbi_float: 57 cases match (actions kinds) | pass |

`scripts/check-test-mapping.py` verifies that every Zig test has exactly one
Go test, in the same order, by the `// Zig: test "<title>"` comment above each
Go test function.

Skips: a Go test skips only where the Zig test returns `error.SkipZigTest` in
the library build.

## How the ported files are tested during the port

A whole-package `go test ./terminal/` stops at the first not-implemented
panic of a file that is not ported yet, so the tests of ported files are run
by name (`-run`). The corpus replay areas (`TestCorpusReplay<Area>`) set up a
`Terminal` and a `Stream` for every case, so they cannot pass before the
stream and terminal layers are ported; until then the parser traces are
compared by `TestCorpusReplaySelfParserTrace`.

## Zig-only lines in the tests

Lines of Zig tests that have no Go counterpart are marked `// Zig-only:` in
the Go test, with the reason, and the rest of the test is kept. When the
implementation gives a Zig-only line a Go counterpart, the line is translated:
with the parser module, `osc.Parser.alloc` (kept as a nil / non-nil field)
made the two allocator lines in the `Parser.zig` tests translatable, and the
colour tokenizer's `next` the two token checks in the `osc/parsers/color.zig`
tests.
