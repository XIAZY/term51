# Pending mapping notes: `std.fmt.parseFloat` port (worktree `pf`)

Sections ready to merge into `docs/mapping.md`. They replace the subsection
"`std.fmt.parseFloat(f64)` (used by `RGB.fromIntensity`)" of the colours
part (see "Corrections" at the end).

## `std.fmt.parseFloat(f64)`: `terminal/std_fmt_parse_float.go`

Zig 0.15.2's `lib/std/fmt/parse_float.zig` and `lib/std/fmt/parse_float/*.zig`
(the toolchain that built the recorded corpus) are ported in full. Nothing in
the file is ported from a Ghostty source file. The only caller is
`RGB.fromIntensity` (`colorRGBFromIntensity`, `color.go`), which parses an
f64.

### Why a port and not strconv

- Grammar: Zig accepts `_` between digits (also in the exponent) without a
  base prefix, a hex mantissa without a `p` exponent (`0x1`) and a signed
  `nan`; strconv rejects those.
- Hex rounding: with more than 16 significant hex digits Zig is not correctly
  rounded (`0x1.000000000000000800000000001p-1` gives 0.5,
  `0xFFFFFFFFFFFFFFFFFFFFp-80` gives 0x3FEFFFFFFFFFFFFE).
- Decimal rounding: Zig's decimal path is **not** always correctly rounded.
  `Decimal.parse` (`decimal.zig` lines 273-294, used by `convertSlow`) trims
  trailing zeros by scanning `s` backwards from `stream.offsetTrue() - 1`.
  `offsetTrue()` does not count the `_` separators that `scanDigit` skipped,
  so with separators the scan starts too far left. If it lands on a run of
  `'0'` characters it counts them, and `num_digits -= n_trailing_zeros` then
  drops that many digits from the end, real digits included. Near a
  midpoint between two doubles this returns the lower one (one unit in the
  last place low). Example: `0.0_66666666666666658802586908905141171999275684356689453125001`
  gives 0x3FB1111111111110; the correctly rounded value (strconv, and Zig
  without the `_`) is 0x3FB1111111111111. Through `rgbi:` that is red 16
  instead of 17. Only inputs that reach the slow path are affected, which
  means more than 19 significant digits near a midpoint, with Eisel-Lemire
  unable to decide.
- Exponents: `Decimal.parse` saturates its exponent at 0x10000 (65,536) and
  `parseScientific` at 0x1000_0000. strconv caps the exponent at 10,000, so
  long inputs with compensating exponents of 100,000 or more differ.

The port copies all of these exactly (governing rule 7).

### Names

- Every declaration has the prefix `stdFmtParseFloat` followed by its Zig
  name in Go style: `parseFloat` -> `stdFmtParseFloatF64`,
  `FloatStream` -> `stdFmtParseFloatFloatStream`, `isEightDigits` ->
  `stdFmtParseFloatIsEightDigits`, file-level constants
  (`max_shift`, `eisel_lemire_smallest_power_of_five`, ...) ->
  `stdFmtParseFloatMaxShift`, `stdFmtParseFloatEiselLemireSmallestPowerOfFive`.
  The prefix names the std file group; unlike the earlier stand-ins
  (`colorStd...`), it carries no `color` prefix: the code is no longer in
  `color.go`, and the name says what it ports, not who calls it.
- Generic instantiations get a type suffix. Generic over the float type T
  (always f64 here): `F64` (`stdFmtParseFloatConvertFastF64`,
  `stdFmtParseFloatNumberF64`, `stdFmtParseFloatBiasedFpF64`,
  `stdFmtParseFloatDecimalF64`, ...). Generic over the mantissa type
  (`mantissaType(f64)` is u64): `U64` (`stdFmtParseFloatTryParseDigitsU64`,
  `stdFmtParseFloatMinNDigitIntU64`, `stdFmtParseFloatFastIntPow10U64`).
- Methods keep Zig's names in lower case (`offsetTrue`, `scanDigit`,
  `leftShift`, `toFloat`, ...): the types are file-private. Functions without
  `self` that return the type follow the `<Type>Init` rule:
  `FloatStream.init` -> `stdFmtParseFloatFloatStreamInit`,
  `BiasedFp.zero`/`zeroPow2`/`inf` -> `stdFmtParseFloatBiasedFpF64Zero` /
  `...ZeroPow2` / `...Inf`, `Decimal.new`/`parse` ->
  `stdFmtParseFloatDecimalF64New` / `...Parse`, `U128.new`/`mul` ->
  `stdFmtParseFloatU128New` / `...Mul`.
- `FloatStream.len` is the method `len` (legal in Go). The local `len` of
  `parsePartialNumberBase` is `len_`.
- std helpers used by the package: `std.ascii.toLower` ->
  `stdFmtParseFloatAsciiToLower`, `std.ascii.startsWithIgnoreCase` ->
  `stdFmtParseFloatAsciiStartsWithIgnoreCase`, `std.math.shl(u64, ...)` /
  `std.math.shr(u64, ...)` -> `stdFmtParseFloatMathShlU64` /
  `stdFmtParseFloatMathShrU64` (signed shift amount: negative reverses the
  direction, 64 or more gives 0, as Zig's), `std.math.nan(f64)` ->
  the existing `colorStdMathNanF64` (`color.go`), `std.math.inf` ->
  `math.Inf`, `std.math.mul(u64)` -> `bits.Mul64` with a non-zero high word
  as the overflow, `@clz` -> `bits.LeadingZeros64`, `std.mem.readInt` /
  `writeInt` (u64, little) -> `encoding/binary.LittleEndian`. All could move
  to `zig.go` later.

### Layout

One Go file for the ten Zig files (the task asked for one file). Each Zig
file is a section headed by its path, functions in the Zig order: 
`parse_float.zig`, `parse.zig`, `FloatStream.zig`, `common.zig`,
`FloatInfo.zig`, `convert_hex.zig`, `convert_fast.zig`,
`convert_eisel_lemire.zig`, `convert_slow.zig`, `decimal.zig`. Upstream's
comments are carried over.

### Comptime

- Only the f64 instantiation exists. The f16, f32, f80 and f128 arms of
  every type switch (`FloatInfo.from`, `fastPow10`, `floatFromUnsigned`,
  `BiasedFp.inf`) and the u128 arm of `fastIntPow10` are not ported;
  `if (MantissaT == u64) a else b` is written as `a` (max_mantissa_digits 19
  and 16, Decimal constants 768/19/2047/-324/310/18). `mantissaType(T)` is
  `uint64` wherever it is used.
- `FloatInfo.from(f64)` (comptime_int fields) is a block of untyped Go
  constants `stdFmtParseFloatFloatInfoF64<Field>`.
- Comptime value parameters become ordinary parameters: `base`, the
  `ParseInfo` struct (`stdFmtParseFloatParseInfo`), `precision` of
  `computeProductApprox`, `n` of `tryParseNDigits`. `comptime assert`s on
  them are run-time `assert`s.
- `math.floatExponentBits(f64)` etc. are literal constants (11, 52, 1023,
  -1022) with the Zig expression in a comment.
- Tables: `eisel_lemire_table_powers_of_five_128` (651 entries) and
  `pow2_to_pow5_table` (128 entries, local to `numberOfDigitsLeftShift` in
  Zig, package-level in Go: `stdFmtParseFloatDecimalPow2ToPow5Table`) were
  copied from the Zig source by a script and compared entry by entry with
  it (identical; every cutoff is also checked to equal 5^i). Note that the
  Eisel-Lemire table passes the high word of each power as `U128.lo` and the
  low word as `.hi`; `computeProductApprox` uses them that way round, and so
  does the port. `fastPow10` and `fastIntPow10` tables are package-level
  arrays (`...Table`) indexed by the function as in Zig (`i & 31`).

### Optionals and errors

- `?T` results are `(T, bool)` (as the previous stand-ins did):
  `FloatStream.first`, `readU64`, `scanDigit`, `parseScientific`,
  `parsePartialNumberBase`, `parsePartialNumber`, `parseNumber`,
  `parsePartialInfOrNan`, `parseInfOrNan`, `convertFast`,
  `convertEiselLemire`.
- `parseFloat`'s only error, `error.InvalidCharacter`, is discarded by the
  only caller, so `stdFmtParseFloatF64` returns `(float64, bool)`; no new
  sentinel.

### Control flow

- `scanDigit`'s `retry: while (true)` with `continue :retry` is a Go `for`
  with `continue`.
- `convertSlow`: the `blk:` block whose `5...9` prong does an unlabelled
  `break` (which leaves the enclosing `while`) is a labelled Go loop
  `shiftLeft:` with `break shiftLeft`.
- `numberOfDigitsLeftShift`: the `return x.delta;` after the if/else chain
  inside the `for` is unreachable in Zig and is omitted (`go vet` rejects
  unreachable code); a comment marks the place.
- `@branchHint(.cold)` has no Go counterpart (comment kept).

### Safety checks (ReleaseSafe recorder)

Where Zig would trap and Go would wrap silently, the port asserts:
`n.mantissa + 1` in `parseFloat`, `n *= 10` in `min_n_digit_int`,
`first.hi += 1` in `computeProductApprox`, `@intCast(self.e)` to u64 in
`BiasedFp.toFloat` (e >= 0), the `@intCast` usize -> i32 conversions and the
`i -= 1` and `d.num_digits -= n_trailing_zeros` subtractions in
`Decimal.parse`. None of them can fail for any input (reasoning in the
comments; the 11.6 million compared inputs never trip them, natively or on 32-bit ARM). Other checked
operations are proven not to overflow by upstream's own comments
(`parse8Digits`) or by bounds (shift amounts, `Decimal` digit arithmetic in
u64 with shifts of at most 60); Go's index checks cover the array accesses.

### Integer widths

`usize` is `uint`, `isize` is `int64` (the recorder is 64-bit; the values are
bounded by the input length), `i64`/`i32`/`u64`/`u8` are the same-width Go
types. `U128.mul` is `bits.Mul64`. Nothing uses Go `int` where the width
matters, so 32-bit ARM computes the same values (checked, below).

### Floating point

The only float arithmetic is in `convertFast`: `float64(mantissa)` (exact,
mantissa <= 2^53), one multiplication or division by a power of ten, and a
negation. There is no addition, so Go cannot fuse anything; `go build
-gcflags=...=-S` of the package shows no FMADD/FMSUB/FNMADD/FNMSUB.
Eisel-Lemire and the slow path are integer-only.

### Not ported as an optimisation

Nothing is left out: the 8-digits-at-a-time path of `tryParseDigits`
(`parse8Digits`, `isEightDigits`, `readU64Unchecked`) and the 8-digit copy
in `Decimal.parse` are ported (the previous stand-in skipped the first one).
Unused declarations (`FloatStream.len`, `isEmpty`, `readU64`, `atUnchecked`,
`BiasedFp.zeroPow2`) are ported too.

### Verification

Compared bit for bit (f64 bits, or ERR) with Zig 0.15.2 `std.fmt.parseFloat(f64)`
using the project lead's harness (copy in `/home/xia/wt/pf-scratch/pfdiff-pf`,
streamed, nothing stored):

| Family | What | Inputs | Zig ERR | Diffs native (arm64) | Diffs 32-bit ARM (qemu) |
|---|---|---:|---:|---:|---:|
| targeted | hand-written edge cases (signs, `inf`/`nan` spellings, hex with/without `p`, underscores valid/invalid, under/overflow) | 123 | 61 | 0 | 0 |
| rgb | the 255 channel boundaries with and without `_`, `rgbi:` domain | 510 | 0 | 0 | 0 |
| bytes | every byte value inserted/substituted at every position of 8 templates | 14,336 | 13,833 | 0 | 0 |
| long | very long inputs (up to 200,000 digits, with and without `_`, decimal and hex) with compensating exponents | 297 | 0 | 0 | 0 |
| intensity | random values in [0, 1] with long digit strings | 200,000 | 63,949 | 0 | 0 |
| random | random decimal and hex strings, random `_` | 300,000 | 24,124 | 0 | 0 |
| halfway | exact midpoints between adjacent doubles, perturbed and truncated, with and without `_` | 1,394,822 | 412,252 | 0 | 0 |
| extra | midpoints with `_` (>19 digits) at the intensity boundaries, huge exponents, hex, inf/nan, malformed | 1,639,789 | 209,253 | 0 | 0 |
| exh6 | every 6-character string over three 11/8-letter alphabets | 3,805,266 | 3,795,432 | 0 | 0 |
| exh5 | every string of up to 5 characters over a 21-letter alphabet | 4,288,305 | 4,275,771 | 0 | 0 |
| **total** | | **11,643,449** | | **0** | **0** |

The Go side is the current `std_fmt_parse_float.go` (package line changed
to `main`), built natively and with `GOARCH=arm GOARM=7`, run under
`qemu-arm-static`; the whole set ran on both. Defect-focused check
(`/home/xia/wt/pf-scratch/defect`): `RGB.parse` built from upstream
`color.zig` with Zig 0.15.2 against `terminal.ColorRGBParse`, natively and
on 32-bit ARM: identical on the 10 `rgbi:` strings of `list.txt` and on all
1,020 generated candidates (`cands.txt`: for each channel boundary k, the
midpoint + `001` with `_` after `0.0`, after `0.0` and two digits later,
near the end, and without `_`); 360 of the 765 underscored candidates give
a channel one lower than the correctly rounded value, none of the 255
without `_`.

Tests: `terminal/std_fmt_parse_float_test.go` translates the six tests of
`parse_float.zig` that have an f64 part (`parseFloat`, `nan and inf`,
`largest normals` (f64 line), `hex.special` and `hex.zero` (written for f32
in Zig, width-independent, run on f64), `hex.f64`); not translated:
`#11169` (f128), `many_digits hex` (f32 vs f128), `hex.f16`, `hex.f32`,
`hex.f80`, `hex.f128`. `parse_float/*.zig` have no tests.
`scripts/check-test-mapping.py` does not check std files (as for
`std_hash_*_test.go`). Two Go-only tests pin the defect bits and the large
exponents. The defect is also pinned through `rgbi:` by the term51 test
`term51: RGB.fromIntensity: underscores can lower a channel by one (zig std
defect)` (term51 patch 0022, translated in `color_test.go`) and by the
recorded suite `osc/rgbi_float` (57 cases: OSC 11, OSC 4 and OSC 21 with
each string, with and without `_`).

## Corrections for the colours section of `docs/mapping.md`

Applied on `impl/pf`: item 1 by a short corrected subsection that points
here (replace it by the section above when folding), item 2 as written.
The stale `color.go` comment (old lines 1196-1200) went with the removed
`colorStdFmtParseFloatF64`; `color.go`'s header now says Zig's result is
not always correctly rounded.

1. Lines 528-547 ("`std.fmt.parseFloat(f64)` (used by `RGB.fromIntensity`)"):
   the sentence at lines 537-539, "Decimal values are converted by
   `strconv.ParseFloat` of the same digits without underscores: Zig's decimal
   path (fast path, Eisel-Lemire, slow path) is correctly rounded, as is
   strconv; `ErrRange` is accepted (it carries Zig's +-inf).", is wrong: Zig's
   decimal path is not correctly rounded for some inputs with `_`
   separators (see above), and strconv also truncates exponents beyond
   10,000. Replace the whole subsection by the section above. Also obsolete
   in it: `colorStdFmtParseFloatF64` (now `stdFmtParseFloatF64`), lines
   540-541 (the 8-digit path is now ported), and lines 542-545 (the 60-input
   check is superseded by the comparison above).
2. Lines 560-563 ("All these stand-ins carry a `color` prefix and live at the
   end of `color.go` ..."): add "except `std.fmt.parseFloat`, which is in
   `std_fmt_parse_float.go` with the prefix `stdFmtParseFloat`".
