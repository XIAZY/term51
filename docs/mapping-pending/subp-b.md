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
  which no Go file had. They are ported in `terminal/os_string_encoding.go`
  (next section): without them 17 translated tests (cmdline,
  cmdline_url, `term51: Command.writeCommandLine: ...`) cannot pass.

### `os/string_encoding.zig` (new file, needs the lead's approval)
- File `terminal/os_string_encoding.go`, prefix `OsStringEncoding`
  (directory in the name, as for subdirectories of src/terminal).
- Ported: `printfQDecode`, `urlPercentDecode` (exported, they are `pub`) and
  `hex` (`osStringEncodingHex`, private). `hex` returns `u4` in Zig; the Go
  result is a uint8 that is always below 16, and `std.math.shl(u8, hex(a),
  4) | hex(b)` is `hex(a)<<4 | hex(b)` on uint8 (no bit is lost).
- `std.ascii.control_code.esc/lf/cr/ht/vt` are the literal bytes with the
  Zig name in a comment.
- Not ported: `isValidChar`, `urlPercentEncode` (only src/synthetic uses
  them; it is not part of libghostty-vt).
- The file's 18 Zig tests (`printf_q 1`..`10`, `singles percent`,
  `percent 1`..`7`) are not translated: they belong in a test file that does
  not exist yet; the lead decides (translate, or list in
  scripts/not-translated.txt).
