// Tests translated from Zig 0.15.2's lib/std/fmt/parse_float.zig (the
// toolchain that built the recorded corpus); the files under
// lib/std/fmt/parse_float/ have no tests. Each Go test carries the exact Zig
// title in a "// Zig: test" comment. scripts/check-test-mapping.py does not
// check this file: it maps Ghostty source files, not std files.
//
// Only the f64 instantiation of std.fmt.parseFloat is ported
// (stdFmtParseFloatF64), so:
//   - `inline for ([_]type{ f16, f32, f64, f80, f128 }) |T|` is its f64
//     iteration;
//   - "largest normals" keeps its f64 line only;
//   - "hex.special" and "hex.zero" are written for f32 only in Zig; their
//     assertions do not depend on the float width, so they are run on f64
//     (checked against Zig 0.15.2 parseFloat(f64): same results);
//   - not translated (no f64 in them): "#11169" (f128), "many_digits hex"
//     (f32 against f128), "hex.f16", "hex.f32", "hex.f80", "hex.f128".
//
// Translation notes:
//   - `try parseFloat(T, s)` is pf_parse(t, s), which stops the test if the
//     string is rejected; `expectError(error.InvalidCharacter,
//     parseFloat(T, s))` is pf_expectInvalid(t, s).
//   - `std.math.approxEqAbs(T, x, y, tolerance)` is pf_approxEqAbs.
//   - `@as(Z, @bitCast(x))` is math.Float64bits(x).
//
// The Go-only tests at the end pin Zig 0.15.2 behaviour that strconv does not
// have (the Decimal.parse trailing-zero defect, exponents beyond 10,000);
// their values come from Zig 0.15.2 std.fmt.parseFloat(f64).

package terminal

import (
	"math"
	"strings"
	"testing"
)

// pf_epsilon is the file-level `const epsilon = 1e-7` of parse_float.zig.
const pf_epsilon = 1e-7

// pf_parse is `try parseFloat(f64, s)`.
func pf_parse(t *testing.T, s string) float64 {
	t.Helper()
	v, ok := stdFmtParseFloatF64([]uint8(s))
	if !ok {
		t.Fatalf("parseFloat(f64, %q): error.InvalidCharacter", s)
	}
	return v
}

// pf_expectInvalid is `try expectError(error.InvalidCharacter,
// parseFloat(f64, s))`.
func pf_expectInvalid(t *testing.T, s string) {
	t.Helper()
	if v, ok := stdFmtParseFloatF64([]uint8(s)); ok {
		t.Fatalf("parseFloat(f64, %q) = %v, want error.InvalidCharacter", s, v)
	}
}

// pf_approxEqAbs is std.math.approxEqAbs(f64, x, y, tolerance).
func pf_approxEqAbs(x float64, y float64, tolerance float64) bool {
	if !(tolerance >= 0) {
		panic("approxEqAbs: tolerance must be >= 0")
	}
	// Fast path for equal values (and signed zeros and infinites).
	if x == y {
		return true
	}
	if math.IsNaN(x) || math.IsNaN(y) {
		return false
	}
	return math.Abs(x-y) <= tolerance
}

// Zig: test parseFloat (lib/std/fmt/parse_float.zig)
func TestStdFmtParseFloat_parseFloat(t *testing.T) {
	// T = f64 (the only ported instantiation)
	pf_expectInvalid(t, "")
	pf_expectInvalid(t, "   1")
	pf_expectInvalid(t, "1abc")
	pf_expectInvalid(t, "+")
	pf_expectInvalid(t, "-")

	expectEqual(t, pf_parse(t, "0"), 0.0)
	expectEqual(t, pf_parse(t, "0"), 0.0)
	expectEqual(t, pf_parse(t, "+0"), 0.0)
	expectEqual(t, pf_parse(t, "-0"), 0.0)

	expectEqual(t, pf_parse(t, "0e0"), 0)
	expectEqual(t, pf_parse(t, "2e3"), 2000.0)
	expectEqual(t, pf_parse(t, "1e0"), 1.0)
	expectEqual(t, pf_parse(t, "-2e3"), -2000.0)
	expectEqual(t, pf_parse(t, "-1e0"), -1.0)
	expectEqual(t, pf_parse(t, "1.234e3"), 1234)

	expect(t, pf_approxEqAbs(pf_parse(t, "3.141"), 3.141, pf_epsilon), `approxEqAbs(T, try parseFloat(T, "3.141"), 3.141, epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, "-3.141"), -3.141, pf_epsilon), `approxEqAbs(T, try parseFloat(T, "-3.141"), -3.141, epsilon)`)

	expectEqual(t, pf_parse(t, "1e-5000"), 0)
	expectEqual(t, pf_parse(t, "1e+5000"), math.Inf(1))

	expectEqual(t, pf_parse(t, "0.4e0066999999999999999999999999999999999999999999999999999"), math.Inf(1))
	expect(t, pf_approxEqAbs(pf_parse(t, "0_1_2_3_4_5_6.7_8_9_0_0_0e0_0_1_0"), float64(123456.789000e10), pf_epsilon), `approxEqAbs(T, try parseFloat(T, "0_1_2_3_4_5_6.7_8_9_0_0_0e0_0_1_0"), @as(T, 123456.789000e10), epsilon)`)

	// underscore rule is simple and reduces to "can only occur between two digits" and multiple are not supported.
	pf_expectInvalid(t, "0123456.789000e_0010")   // cannot occur immediately after exponent
	pf_expectInvalid(t, "_0123456.789000e0010")   // cannot occur before any digits
	pf_expectInvalid(t, "0__123456.789000e_0010") // cannot occur twice in a row
	pf_expectInvalid(t, "0123456_.789000e0010")   // cannot occur before decimal point
	pf_expectInvalid(t, "0123456.789000e0010_")   // cannot occur at end of number

	expect(t, pf_approxEqAbs(pf_parse(t, "1e-2"), 0.01, pf_epsilon), `approxEqAbs(T, try parseFloat(T, "1e-2"), 0.01, epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, "1234e-2"), 12.34, pf_epsilon), `approxEqAbs(T, try parseFloat(T, "1234e-2"), 12.34, epsilon)`)

	expect(t, pf_approxEqAbs(pf_parse(t, "1."), 1, pf_epsilon), `approxEqAbs(T, try parseFloat(T, "1."), 1, epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, "0."), 0, pf_epsilon), `approxEqAbs(T, try parseFloat(T, "0."), 0, epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, ".1"), 0.1, pf_epsilon), `approxEqAbs(T, try parseFloat(T, ".1"), 0.1, epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, ".0"), 0, pf_epsilon), `approxEqAbs(T, try parseFloat(T, ".0"), 0, epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, ".1e-1"), 0.01, pf_epsilon), `approxEqAbs(T, try parseFloat(T, ".1e-1"), 0.01, epsilon)`)

	pf_expectInvalid(t, ".")   // At least one digit is required.
	pf_expectInvalid(t, ".e1") // At least one digit is required.
	pf_expectInvalid(t, "0.e") // At least one digit is required.

	expect(t, pf_approxEqAbs(pf_parse(t, "123142.1"), 123142.1, pf_epsilon), `approxEqAbs(T, try parseFloat(T, "123142.1"), 123142.1, epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, "-123142.1124"), float64(-123142.1124), pf_epsilon), `approxEqAbs(T, try parseFloat(T, "-123142.1124"), @as(T, -123142.1124), epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, "0.7062146892655368"), float64(0.7062146892655368), pf_epsilon), `approxEqAbs(T, try parseFloat(T, "0.7062146892655368"), @as(T, 0.7062146892655368), epsilon)`)
	expect(t, pf_approxEqAbs(pf_parse(t, "2.71828182845904523536"), float64(2.718281828459045), pf_epsilon), `approxEqAbs(T, try parseFloat(T, "2.71828182845904523536"), @as(T, 2.718281828459045), epsilon)`)
}

// Zig: test "nan and inf" (lib/std/fmt/parse_float.zig)
func TestStdFmtParseFloat_nan_and_inf(t *testing.T) {
	// T = f64, Z = u64
	expectEqual(t, math.Float64bits(pf_parse(t, "nAn")), math.Float64bits(colorStdMathNanF64()))
	expectEqual(t, pf_parse(t, "inF"), math.Inf(1))
	expectEqual(t, pf_parse(t, "-INF"), -math.Inf(1))
}

// Zig: test "largest normals" (lib/std/fmt/parse_float.zig)
func TestStdFmtParseFloat_largest_normals(t *testing.T) {
	// Only the f64 line; the f16, f32, f80 and f128 lines are not ported.
	expectEqual(t, math.Float64bits(pf_parse(t, "1.7976931348623157E308")), 0x7fef_ffff_ffff_ffff)
}

// Zig: test "hex.special" (lib/std/fmt/parse_float.zig)
func TestStdFmtParseFloat_hex_special(t *testing.T) {
	// f32 in Zig, f64 here (see the header).
	expect(t, math.IsNaN(pf_parse(t, "nAn")), `math.isNan(try parseFloat(f32, "nAn"))`)
	expect(t, math.IsInf(pf_parse(t, "iNf"), 1), `math.isPositiveInf(try parseFloat(f32, "iNf"))`)
	expect(t, math.IsInf(pf_parse(t, "+Inf"), 1), `math.isPositiveInf(try parseFloat(f32, "+Inf"))`)
	expect(t, math.IsInf(pf_parse(t, "-iNf"), -1), `math.isNegativeInf(try parseFloat(f32, "-iNf"))`)

	expect(t, math.IsInf(pf_parse(t, "0x9999p9999"), 1), `math.isPositiveInf(try parseFloat(f32, "0x9999p9999"))`)
	expect(t, math.IsInf(pf_parse(t, "-0x9999p9999"), -1), `math.isNegativeInf(try parseFloat(f32, "-0x9999p9999"))`)
}

// Zig: test "hex.zero" (lib/std/fmt/parse_float.zig)
func TestStdFmtParseFloat_hex_zero(t *testing.T) {
	// f32 in Zig, f64 here (see the header). 0.0 == -0.0, as in Zig.
	expectEqual(t, float64(0.0), pf_parse(t, "0x0"))
	expectEqual(t, float64(0.0), pf_parse(t, "-0x0"))
	expectEqual(t, float64(0.0), pf_parse(t, "0x0p42"))
	expectEqual(t, float64(0.0), pf_parse(t, "-0x0.00000p42"))
	expectEqual(t, float64(0.0), pf_parse(t, "0x0.00000p666"))
}

// Zig: test "hex.f64" (lib/std/fmt/parse_float.zig)
func TestStdFmtParseFloat_hex_f64(t *testing.T) {
	expectEqual(t, pf_parse(t, "0x1p0"), 1.0)
	expectEqual(t, pf_parse(t, "-0x1p-1"), -0.5)
	expectEqual(t, pf_parse(t, "0x10p+10"), 16384.0)
	expectEqual(t, pf_parse(t, "0x10p-10"), 0.015625)
	// Max normalized value.
	expectEqual(t, pf_parse(t, "0x1.fffffffffffffp+1023"), math.MaxFloat64)
	expectEqual(t, pf_parse(t, "-0x1.fffffffffffffp1023"), -math.MaxFloat64)
	// Min normalized value.
	expectEqual(t, pf_parse(t, "0x1p-1022"), 0x1p-1022)
	expectEqual(t, pf_parse(t, "-0x1p-1022"), -0x1p-1022)
	// Min denormalized value.
	expectEqual(t, pf_parse(t, "0x1p-1074"), math.SmallestNonzeroFloat64)
	expectEqual(t, pf_parse(t, "-0x1p-1074"), -math.SmallestNonzeroFloat64)
}

// Go-only: reproduces Zig 0.15.2 std defect: Decimal.parse (decimal.zig
// 273-294, reached through convertSlow) trims trailing zeros starting from
// stream.offsetTrue() - 1, which does not count the '_' separators, so with
// separators it can count real '0' digits as trailing zeros and drop them;
// near a midpoint the result is one unit in the last place low. Expected bits
// are Zig 0.15.2 std.fmt.parseFloat(f64); the same digits without '_' give
// the correctly rounded value (strconv agrees with those).
func TestStdFmtParseFloat_GoOnly_reproduces_Zig_0_15_2_std_defect_underscores_in_Decimal_parse(t *testing.T) {
	for _, c := range []struct {
		s          string
		bits       uint64 // Zig 0.15.2, with the '_' separators
		bitsNoUndr uint64 // Zig 0.15.2, the same string without '_' (correctly rounded)
	}{
		{"0.0_66666666666666658802586908905141171999275684356689453125001", 0x3fb1111111111110, 0x3fb1111111111111},
		{"0.2_62_7450980392156687504012779754702933132648468017578125001", 0x3fd0d0d0d0d0d0d0, 0x3fd0d0d0d0d0d0d1},
		{"0.5_21568627450980371040856198305846191942691802978515625001", 0x3fe0b0b0b0b0b0b0, 0x3fe0b0b0b0b0b0b1},
		{"0.98039215686274500116681451800104696303606033325195312_5001", 0x3fef5f5f5f5f5f5e, 0x3fef5f5f5f5f5f5f},
		{"0.9_60784313725490168867082729775574989616870880126953125001", 0x3feebebebebebebe, 0x3feebebebebebebf},
		{"6.6_666666666666658802586908905141171999275684356689453125001e-2", 0x3fb1111111111110, 0x3fb1111111111111},
		// The '_' does not always matter: this one is correctly rounded.
		{"0.0_70588235294117639140321074364692321978509426116943359375001", 0x3fb2121212121212, 0x3fb2121212121212},
	} {
		expectEqual(t, c.bits, math.Float64bits(pf_parse(t, c.s)))
		expectEqual(t, c.bitsNoUndr, math.Float64bits(pf_parse(t, strings.ReplaceAll(c.s, "_", ""))))
	}
}

// Go-only: Zig 0.15.2 std.fmt.parseFloat(f64) accepts decimal exponents far
// beyond strconv's cap of 10,000 when the digits compensate for them (Zig
// returns 1.0 for both; strconv.ParseFloat returns +Inf and 0).
func TestStdFmtParseFloat_GoOnly_large_compensated_exponents(t *testing.T) {
	expectEqual(t, uint64(0x3ff0000000000000), math.Float64bits(pf_parse(t, "1"+strings.Repeat("0", 100000)+"e-100000")))
	expectEqual(t, uint64(0x3ff0000000000000), math.Float64bits(pf_parse(t, "0."+strings.Repeat("0", 99999)+"1e100000")))
}
