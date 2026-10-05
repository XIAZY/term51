# Mapping notes from impl/subp-b (OSC sub-parsers)

Files: `osc_parsers_osc9.go`, `osc_parsers_semantic_prompt.go`, and the new
`os_string_encoding.go` (see below).

## Language features (additions)

| Zig | Go | Notes |
|---|---|---|
| Labeled block `blk: { ... break :blk; ... }` that is left from inside nested switches | Go block `{ ... goto blk ... }` followed by the label `blk:` | Go's `break` cannot leave a plain block, and from inside a nested `switch` it only leaves the switch. The label sits where `break :blk` lands; nothing is declared between a `goto` and its label. Used for `conemu:` (osc9.parse), `valid:` (semantic_prompt.parse), `value:` (Option.read) and `data:` (printfQDecode). |
| Labeled block that is a whole switch prong (`'A' => fresh_line: { ... break :fresh_line; ... }`, `progress: { ... }`) | `break` out of the Go `switch` | Equivalent because nothing follows the block inside the prong. The Zig label is kept as a comment. |
| `std.fmt.parseUnsigned` / `std.fmt.parseInt` with base 10, error discarded (`catch null`, `else \|_\|`) | a file-private port that returns `(value, ok)` | No Go function accepts the same inputs (Zig allows `_` between digits; parseUnsigned rejects a sign). Returning `ok` instead of an error avoids new sentinels (`ErrInvalidCharacter`, `ErrOverflow` would have to join `zigSentinels`); every caller discards the error. `oscParsersOsc9ParseUnsigned(buf, bits)` (u16 and usize) and `oscParsersSemanticPromptParseInt(buf)` (i32). subp-a has its own (`oscParsersColorParseIntU9`, ...); these could later be merged into one zig.go helper. |
| `usize` as the result type of a parse whose overflow is observable | 64 bits on every target | osc9 `9;4;1;<n>` parses into usize: on a 32-bit target Zig would reject 2^32..2^64-1, the recorded corpus (64-bit) clamps them to 100. The tests (`term51: parse: progress value`) expect the 64-bit behaviour, and pass under qemu-arm. |

## Per-file notes

### `osc/parsers/osc9.zig`
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

### `osc/parsers/semantic_prompt.zig`
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

### `os/string_encoding.zig` (new file, approved by the lead)
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
