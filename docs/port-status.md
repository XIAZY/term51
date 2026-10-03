# Port status

One row per Zig file. A file is "ported" when its Go file exists, every
upstream test and every project test for it is translated and passes, and the
recorded cases that exercise it pass, on this machine and as a 32-bit ARM
build.

| Zig file | Go file | Code | Zig tests translated | Corpus | 32-bit ARM |
|---|---|---|---|---|---|
| `terminal/parse_table.zig` | `terminal/parse_table.go` | done | 20 of 20 (1 upstream, 19 project), all pass | table equals upstream's (3,584 entries) | pass |
| `terminal/UTF8Decoder.zig` | `terminal/UTF8Decoder.go` | done | 15 of 15 (3 upstream, 12 project), all pass | 69,207 of 69,207 decode cases | pass |
| `terminal/Parser.zig` | `terminal/Parser.go` | done except OSC | 73 of 73 translated (25 upstream, 48 project); 67 pass, 6 need OSC and are behind the `osc_ported` tag | 8,593 of 8,984 parser traces; 391 pending on OSC | pass |
| `terminal/osc.zig` | `terminal/osc.go` | **placeholder** | - | - | - |

`scripts/check-test-mapping.py` verifies that every Zig test has exactly one
Go test, in the same order, by the `// Zig: test "<title>"` comment above each
Go test function.

## Pending

- `terminal/osc.go` is a placeholder so that `Parser.go` compiles. It is
  replaced when `osc.zig` is signed off for porting. Until then:
  - 391 recorded parser traces that contain an OSC dispatch are not compared
    (`TestCorpusParserTrace` counts and logs them).
  - Six Parser tests need a working OSC parser. They are translated in
    `terminal/Parser_osc_test.go` behind the build tag `osc_ported` and are
    not compiled by default. Lines marked `TODO(osc)` use names the real
    `osc.go` has yet to define.
    - `osc: change window title`
    - `osc: change window title (end in esc)`
    - `osc: 112 incomplete sequence`
    - `osc: 104 empty`
    - `term51: next: exit, transition and entry actions occupy slots 0, 1 and 2`
    - `term51: doAction: osc_put feeds the OSC parser`

## Zig-only assertions not translated

Lines of Zig tests that have no Go counterpart. Each is marked
`// Zig-only:` in the Go test, and the rest of the test is kept.

| Test | Zig line | Reason |
|---|---|---|
| `term51: init: starts in ground with empty accumulators` | `try testing.expect(p.osc_parser.alloc == null);` | allocator parameters are dropped in the port |
| `term51: deinit: releases an OSC buffer that was never terminated` | `p.osc_parser.alloc = testing.allocator;` | the leak check it enables has no Go counterpart |
| `osc: 112 incomplete sequence` | `p.osc_parser.alloc = std.testing.allocator;` | same |
| `osc: 104 empty` | `p.osc_parser.alloc = std.testing.allocator;` | same |
