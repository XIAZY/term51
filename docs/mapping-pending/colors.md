# Mapping notes from the colours port (worktree `colors`)

Files: `sgr.go`, `color.go`, `x11_color.go`, `kitty_color.go`,
`mouse_shape.go`, plus `terminal/res/rgb.txt`. Sections below are ready to
merge into `docs/mapping.md`.

## Per-file notes

### `sgr.zig`
- `Parser.next`: the `underline:` labelled block. Go cannot `break` out of a
  labelled block; every `break :underline` leaves the block and then the
  `switch`, which is exactly what a plain Go `break` inside `case 4:` does
  (the code after the switch returns `.unknown`).
- `parseDirectColor` takes the tag as a run-time value; `@unionInit(Attribute,
  @tagName(tag), rgb)` is the helper `sgrAttributeInitRGB` (the three RGB
  variants; any other tag panics "unreachable", Zig's compile error).
- `countColon`: `self.params.len - 1` traps in Zig on an empty list; Go
  asserts `len(params) > 0` (never false: every caller has read a param).
- `@truncate` of a `u16` param to `u8` is the Go conversion `uint8(...)`.
- `@branchHint` has no Go counterpart; the comments next to them are kept.
- The separator list is `StaticBitSet24`, whose `isSet` asserts `index < 24`
  as Zig's does; the colon-skipping loop in `next` therefore stops the program
  where upstream would.

### `color.zig`
- `color.default` and `DynamicPalette.default` are package variables
  initialised at start-up (`colorDefaultInit`, a function holding the comptime
  block; `colorDefaultCubeLevel` is the `if (c == 0) 0 else c*40+55`
  expression). The gray-ramp loop keeps `i +%= 1` on a `uint8` (wraps to 0).
- `std.StaticBitSet(256)` is `StaticBitSet256` (in `color.go`, to move to
  `zig.go`): four `uint64` masks, as Zig's `ArrayBitSet(usize, 256)`; `isSet`,
  `set`, `unset` assert `index < 256`. The iterator reads the live set at each
  step (Zig copies the current word); they differ only if the set changes
  during iteration, which no code does.
- `Name.String()` returns `""` for 16..255 as the wireframe (commit dde4530)
  says; dumps print `#N`. Note: mapping.md's section "Enums that are not
  exhaustive print `#N`" says `String()` should return `#N` itself. The dump
  output is the same either way (`dumpTagName` treats `""` as unnamed); the
  lead should pick one and fix the other text.
- `DynamicRGB.default` is the field `default_` (Go keyword rule).
- `RGB.fromIntensity` includes the term51 reference fix
  (`patches-reference/0002`, upstream-findings section 11): NaN returns
  `ErrInvalidFormat` (upstream aborts in `@intFromFloat`).
- `RGB.parse`: the `RGB{ .r = try fromHex(..), ... }` literals of the `#`
  forms are the helper `colorRGBFromHex3` (same order, first error wins).
- `fromHex`: `@intCast` to `u8` is an explicit `assert(v <= 255)`.
- `LAB.toRgb`: `@max`/`@min` on f32 return the non-NaN operand (LLVM
  maxnum/minnum); Go's built-ins return NaN, so `colorZigMaxF32` /
  `colorZigMinF32` are used. `@intFromFloat` truncates (Go conversion).

### Floating point (all of `color.zig`)
- Fused multiply-add: Go may fuse `a*b + c` (it does on arm64); Zig/LLVM
  does not. Every product that is an operand of `+` or `-` is wrapped in an
  explicit `float32(...)`/`float64(...)` conversion (Go spec: an explicit
  conversion rounds and prevents fusion). Applied in `LAB.*`, `luminance`,
  `perceivedLuminance` and in every stand-in below.
- `std.math.pow(f32)`, `std.math.pow(f64)`, `std.math.cbrt(f32)`,
  `std.math.scalbn`/`ldexp` (f32, f64) are ported (`colorStdMathPowF32`,
  `colorStdMathPowF64`, `colorStdMathCbrtF32`, `colorStdMathLdexpF32/F64`).
  `@exp`/`@log` call compiler_rt's `expf`/`logf`/`exp`/`log` (musl-derived)
  in the recorder's build; those are ported too (`colorCompilerRtExpf`,
  `colorCompilerRtLogf`, `colorCompilerRtExp`, `colorCompilerRtLog`).
  Evidence: with `float32(math.Pow(float64(x), float64(y)))` instead of the
  port, `data/colors/lab_grid.txt` fails at line 9, `lab_lerp.txt` at line 2
  and `generate256.txt` fails. With the port all three match bit for bit,
  natively (arm64) and under 32-bit ARM qemu. (Go's `math.Cbrt` happened to
  match on the grid; the port is kept for fidelity.)
- `math.frexp` uses Go's `math.Frexp` on the value widened to float64
  (exact: same significand and exponent). `@trunc` (modf) uses `math.Trunc`
  (exact). `@sqrt` on f32 is `float32(math.Sqrt(float64(x)))`, the correctly
  rounded f32 root, as LLVM's.
- `std.math.nan(T)` is the quiet NaN `0x7FF8000000000000` / `0x7FC00000`
  (`colorStdMathNanF64/F32`); Go's `math.NaN()` has payload ...0001.
- Additional check run once during the port (not committed): Zig 0.15.2
  `std.math.pow(f64)` on 256x3 inputs and `pow(f32)`/`cbrt(f32)` on 2000x4
  inputs compared bit for bit with the Go stand-ins: identical natively and
  on 32-bit ARM.

### `std.fmt.parseFloat(f64)` (used by `RGB.fromIntensity`)
- strconv's grammar is not Zig's: Zig accepts `_` between digits (also in the
  exponent) without a base prefix, a hex mantissa without `p` (`0x1`), and a
  signed `nan`; strconv rejects those. `colorStdFmtParseFloatF64` ports
  `parse_float.zig` + `parse.zig` (grammar, `Number`, `validUnderscores`,
  `parseInfOrNan`) and `convert_hex.zig` (hex conversion, including Zig's
  handling of more than 16 significant hex digits, which is not correctly
  rounded: `0x1.000000000000000800000000001p-1` gives 0.5,
  `0xFFFFFFFFFFFFFFFFFFFFp-80` gives 0x3FEFFFFFFFFFFFFE). Decimal values are
  converted by `strconv.ParseFloat` of the same digits without underscores:
  Zig's decimal path (fast path, Eisel-Lemire, slow path) is correctly rounded,
  as is strconv; `ErrRange` is accepted (it carries Zig's +-inf).
- The 8-digits-at-a-time fast path of `tryParseDigits` is not ported (same
  result, wrapping arithmetic).
- Checked once against Zig 0.15.2 on 60 edge inputs (signs, `.5`, `1.`,
  underscores valid and invalid, hex with/without `p`, inf/nan spellings,
  overflow/underflow, 20-50 digit decimals, >16-digit hex): identical bits
  and identical accept/reject.
- The error (`error.InvalidCharacter`) is discarded by the only caller, so
  the Go function returns `(float64, bool)`; no new sentinel.

### `std.fmt.parseUnsigned` / `parseInt`
- `colorStdFmtParseUnsigned(buf, base, bits)` follows `parseIntWithSign`
  (`.pos`): no sign, `_` skipped but not first/last (consecutive `_` are
  accepted), accumulation in max(8, bits) bits with overflow as an error.
  Used by `RGB.fromHex` (u16, base 16), `Kind.parse` (u8, base 10) and
  `colorMap` (via `x11ColorParseIntU8`, which adds `parseInt`'s optional
  sign). Callers discard the error kind, so it returns `(uint64, bool)` and
  declares no `ErrInvalidCharacter`/`ErrOverflow` (avoids duplicates with the
  OSC port, which may declare them).
- `std.mem.trim` and `std.mem.indexOfScalarPos` are `colorStdMemTrim` and
  `colorStdMemIndexOfScalarPos`.
- All these stand-ins carry a `color` prefix and live at the end of
  `color.go` to avoid name clashes with stand-ins the parallel OSC port may
  add; they belong in `zig.go` and should be unified with the OSC port's
  equivalents when merging.

### `x11_color.zig`
- `@embedFile("res/rgb.txt")` is `//go:embed res/rgb.txt` of a byte-for-byte
  copy of upstream's `src/terminal/res/rgb.txt` at `terminal/res/rgb.txt`
  (`TestX11Color_corpus_rgb_txt` compares it with the pinned upstream file).
- `map` (comptime `StaticStringMapWithEql(RGB, eqlAsciiIgnoreCase)`) is built
  at start-up by `colorMap`; a parse error (a compile error in Zig) panics.
  The Go map is keyed by the name with ASCII letters lower-cased
  (`x11ColorAsciiLower`); `get` lower-cases its argument the same way, which
  is `eqlAsciiIgnoreCase`'s comparison (non-ASCII bytes compare exactly). The
  782 names are distinct ignoring ASCII case, so no entry hides another.

### `kitty/color.zig`
- `OSC.deinit` sets `list` to nil (`ArrayList.deinit` leaves it undefined).
- `Kind.parse`: `std.meta.stringToEnum(Special, key)` is an exact
  comparison with the tag names.
- `Kind.format` returns the decimal palette index or the tag name.

### `mouse_shape.zig`
- `string_map` (`StaticStringMap`, exact match, distinct keys) is a Go map
  literal with upstream's 56 entries in upstream's order.
- `MouseShape.String` and the other exhaustive enums' `String()` index a
  names array; an out-of-range value (impossible in Zig) panics.
