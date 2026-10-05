### `stream.zig`

File: `terminal/stream.go`. The design of the generic, the handler interface
and `Action` is the wireframe's (header of `stream.go`); unchanged.

#### Language features (additions)

| Zig | Go | Notes |
|---|---|---|
| `try self.handler.vt(.key, payload)` | `if err := self.handler.Vt(StreamActionKeyKey, StreamAction{Tag: StreamActionKeyKey, Key: payload}); err != nil { return err }` | the union is built in place at every call; no helper. A void payload is `StreamAction{Tag: ...}` |
| `@hasDecl(T, "vtRaw")` (comptime) | `any(self.handler).(StreamHandlerVtRaw)` (run-time type assertion) | `vtRaw(...) catch \|err\| err: { log.warn; break :err true }` is: on a non-nil error, warn and `skip = true` |
| Labeled block that is a whole prong of the outer `switch (input.final)` / `switch (action.final)` and is left from inside a nested switch (`mode:`, `decrqm:`, `capture:`, `push:`, `set:`, `decsasd:`, `blk:`) | the outer Go switch carries the label `prong:` and `break :x` is `break prong` | equivalent because nothing follows the outer switch in csiDispatch/escDispatch. Each site has a `// break :x` comment |
| Value-producing inner block (`ansi:`, `question:` in `h`, `l`, `n`) | if / else if / else assigning the value; the failing branch does what Zig's does (`break prong` or `return nil`) | |
| `ansi:` of DECRQM (`p`), left from inside a nested switch | `goto ansi`, label after the block | the mapping's existing goto rule |
| `csi_param:` labeled `if` in nextNonUtf8, left with `break :csi_param` from the `else` of a switch | `goto csi_param` to a label after the `if` | same rule |
| `std.meta.intToEnum(csi.EraseDisplay, p) catch null` (exhaustive enum) | switch on the u16 value over the five enum values | the check is on the u16, before any conversion, so a value above 255 cannot be truncated into a valid one |
| `std.meta.intToEnum(csi.TabClear, p) catch` (non-exhaustive `enum(u8)`) | `if p > 0xFF { error branch }` | `std.enums.fromInt` of a non-exhaustive enum only checks that the value fits the tag type |
| `std.math.cast(u5, p) orelse` | `if p > 0x1F` | |
| `inline 22, 23 => \|number\|` | `case 22, 23:` with `number := input.Params[0]`; the `else => @compileError` arm is `panic("unreachable")` | |
| `@max(1, p)` | Go builtin `max(1, p)` on uint16 | |
| `std.unicode.utf8ValidateSlice` | `utf8.Valid` | same equivalence as `isSafeUtf8` (osc/encoding.zig section): both reject overlongs, surrogates and values above U+10FFFF |
| Exhaustive `switch` over an enum / union (`Parser.Action`, `osc.Command`, `csi.EraseDisplay`, `ansi.ProtectedMode`, `ansi.ModifyKeyFormat`, the action tag of `CSI = u`) | `default: panic("unreachable")` | as in osc.go |
| Non-exhaustive `switch` with `_ =>` (`csi.EraseLine`, `csi.TabClear`) | `default:` | |
| `{}` of a value of a non-exhaustive enum in a log line | `@enumFromInt(%d)` | Zig 0.15.2 `printEnumNonexhaustive` prints `.name` for a named value and `@enumFromInt(N)` otherwise; both `_` prongs are only reached by unnamed values |

#### Logging

- `log.warn` is `warn("stream", ...)`. `log.err` (the one in `CSI t` with no
  parameters) also goes through `warn("stream", ...)`, marked `// Zig: log.err`:
  the package has one hook, and an error-level message is printed whenever a
  warning is. The lead may prefer a separate hook.
- `log.info` (C1 execute, change icon, the `debug`-only lines) and
  `log.debug` (unimplemented OSC callbacks) are not printed; the calls are
  kept as comments, as in `search_screen.go` and `osc_parsers_iterm2.go`.
- Formats: `{f}` is `Format()` with `%s` (`ParserActionCSI`, `ParserActionESC`,
  `ParserAction`); `{s}` of the intermediates is `%s` of the `[]uint8`; `{}`
  of an integer is `%d`; `{any}` of the intermediates (configureCharset) is
  `zigFmtAnySlice`; `{}` of an error is `error.%v`; `0x{x}` is `0x%x`.

#### `nextSlice`, `nextSliceCapped` and the decode stand-in

- `nextSlice` is the scalar loop only: `comptime debug or !build_options.simd`
  is true for the reference build. `streamDebug` (Zig `debug`) is declared but
  only referred to in comments (its two uses are `log.info`).
- `nextSliceCapped` is ported line for line (the tests call it directly). Its
  call `simd.vt.utf8DecodeUntilControlSeq` is the file-private stand-in
  `streamUtf8DecodeUntilControlSeq` (result `streamDecodeResult{consumed,
  decoded}`, simd/vt.zig's `DecodeResult`). Upstream's scalar fallback
  `utf8DecodeUntilControlSeqScalar` is NOT ported, because it is the cause of
  the abort on `FF ESC [ 1 A` (stream.zig:593, upstream-findings section 12)
  and of the `E2 82 (` disagreement with the per-byte path; docs/mapping.md
  requires nextSliceCapped to return what nextSlice returns.
- The stand-in decodes the bytes before the first ESC with a fresh
  `UTF8Decoder`, feeding a byte again when the decoder does not consume it
  (as nextUtf8 does). `consumed` ends after the last complete or rejected
  sequence; a sequence still open at the end of the input is left for
  nextSliceCapped's `nextUtf8` loop (it stays pending in the stream's
  decoder); a sequence still open when the ESC is reached gets the U+FFFD the
  decoder emits on rejecting the ESC, and the bytes up to the ESC are
  consumed, so the ESC is handled by `consumeAllEscapes` as `handleCodepoint`
  would. This keeps nextSliceCapped's control flow and makes it identical to
  the per-byte path: checked once (not committed) on 30,000 random inputs of
  0-23 bytes over an alphabet of ESC, CSI/OSC bytes, C0, invalid and
  truncated UTF-8 leads, continuation bytes, surrogate and overlong starts,
  each fed in random chunks: same actions, parser state and decoder state as
  byte-by-byte `next`.

#### Other per-function notes

- `Execute`: `ansi.C0` is `enum(u7)` non-exhaustive; `ANSIC0(c)` after the
  `c > 0x7F` check. The C1 path builds `ParserActionESC{Intermediates:
  []uint8{}, ...}` (Zig `&.{}`).
- nextNonUtf8's CSI-param fast path: `*|=` / `+|=` are `saturatingMulU16` /
  `saturatingAddU16`; `params_sep.set(int(params_idx))`.
- `Print` takes `uint32` (u21); `handleCodepoint`'s `@intCast(c)` to u8 for
  `execute` is a plain conversion after `c <= 0xF`.
- Payload widths: the usize payloads (`print_repeat`, `delete_chars`, ...)
  are `uint(input.Params[0])`.
- `Stream.init` copies Zig's whole doc comment (the OSC 52 example and the
  size-limit sentence were missing from the wireframe).
