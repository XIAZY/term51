# Port status

One row per Zig file. A file is "ported" when its Go file exists, every
upstream test and every project test for it is translated and passes, and the
recorded cases that exercise it pass, on this machine and as a 32-bit ARM
build.

| Zig file | Go file | Code | Upstream tests | Project tests | Corpus | 32-bit ARM |
|---|---|---|---|---|---|---|
| `terminal/parse_table.zig` | `terminal/parse_table.go` | done | pending translation | pending translation | table equals upstream's (3,584 entries) | pass |
| `terminal/UTF8Decoder.zig` | `terminal/UTF8Decoder.go` | done | pending translation | pending translation | 69,207 of 69,207 decode cases | pass |
| `terminal/Parser.zig` | `terminal/Parser.go` | done except OSC | pending translation | pending translation | 8,593 of 8,984 parser traces; 391 pending on OSC | pass |
| `terminal/osc.zig` | `terminal/osc.go` | **placeholder** | - | - | - | - |

## Pending

- `terminal/osc.go` is a placeholder so that `Parser.go` compiles. It is
  replaced when `osc.zig` is signed off for porting. Until then:
  - 391 recorded parser traces that contain an OSC dispatch are not compared
    (`TestCorpusParserTrace` counts and logs them).
  - Parser tests that need a working OSC parser are kept behind the build tag
    `osc_ported` and listed here when translated.
