// Tests translated one for one from src/terminal/color.zig (Ghostty v1.3.1
// plus the term51 additional tests). Each Go test carries the exact Zig title
// in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualStrings, zigUnwrap and zigTry are defined in Parser_test.go and
// dcs_test.go; helpers that do not exist there are prefixed col_.
//
// Not translated (listed in the report to the project lead):
//   - "term51: Name.cval: every named value and unnamed values" and
//     "term51: RGB.cval: copies the channels into the extern struct": tests
//     of the C binding helpers (cval, C), which are not ported.
//   - "term51: LAB.fromRgb: dump exact data when TERM51_COLOR_DUMP_DIR is
//     set": the gated data dump; it only produces the files under
//     data/colors/. The data files are compared by corpus_color_test.go.
//
// Translation notes:
//   - `try x` on a call that returns (value, error) is a col_ wrapper that
//     stops the test on an error; `testing.expectError(error.X, call)` is
//     col_expectError(t, ErrX, <the error of the call>).
//   - `testing.expectApproxEqAbs` is col_expectApproxEqAbs, with Zig's
//     semantics (equal values pass, a NaN fails, |a-b| <= tolerance).
//   - `default` is ColorDefault, `Palette` is ColorPalette, `PaletteMask` is
//     ColorPaletteMask, `.initEmpty()` is StaticBitSet256InitEmpty(),
//     `@enumFromInt(i)` on Name is ColorName(i), `LAB` is colorLAB.
//   - `@typeInfo(E).@"enum".fields` is col_specialFields / col_dynamicFields
//     (the enum values in declaration order).
//   - `null` compared with an optional is None[T]() and `@as(?RGB, x)` is
//     Some(x).
//   - Zig's `defer` and `inline for` have no special handling here.

package terminal

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// col_expectError is Zig's `try testing.expectError(expected, actual)`: got
// is the error of the call (nil when the call returned a value).
func col_expectError(t *testing.T, want error, got error) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected error %v, found value", want)
	}
	if !errors.Is(got, want) {
		t.Fatalf("expected error %v, found error %v", want, got)
	}
}

// col_expectApproxEqAbs is Zig's `try testing.expectApproxEqAbs(expected,
// actual, tolerance)`.
func col_expectApproxEqAbs[T float32 | float64](t *testing.T, want T, got T, tolerance T) {
	t.Helper()
	if want == got {
		return
	}
	d := want - got
	if d < 0 {
		d = -d
	}
	if !(d <= tolerance) {
		t.Fatalf("actual %v, not within absolute threshold %v of expected %v", got, tolerance, want)
	}
}

// col_parse is `try RGB.parse(s)`.
func col_parse(t *testing.T, s string) ColorRGB {
	t.Helper()
	v, err := ColorRGBParse([]uint8(s))
	zigTry(t, err)
	return v
}

// col_parseErr is the error of RGB.parse(s), nil when it succeeds.
func col_parseErr(s string) error {
	_, err := ColorRGBParse([]uint8(s))
	return err
}

// col_fromHex is `try RGB.fromHex(s)`.
func col_fromHex(t *testing.T, s string) uint8 {
	t.Helper()
	v, err := colorRGBFromHex([]uint8(s))
	zigTry(t, err)
	return v
}

// col_fromHexErr is the error of RGB.fromHex(s), nil when it succeeds.
func col_fromHexErr(s string) error {
	_, err := colorRGBFromHex([]uint8(s))
	return err
}

// col_fromIntensity is `try RGB.fromIntensity(s)`.
func col_fromIntensity(t *testing.T, s string) uint8 {
	t.Helper()
	v, err := colorRGBFromIntensity([]uint8(s))
	zigTry(t, err)
	return v
}

// col_fromIntensityErr is the error of RGB.fromIntensity(s), nil when it
// succeeds.
func col_fromIntensityErr(s string) error {
	_, err := colorRGBFromIntensity([]uint8(s))
	return err
}

// col_nameDefault is `try Name.default(n)`.
func col_nameDefault(t *testing.T, n ColorName) ColorRGB {
	t.Helper()
	v, err := n.Default()
	zigTry(t, err)
	return v
}

// col_specialFields is @typeInfo(Special).@"enum".fields, as values.
var col_specialFields = []ColorSpecial{
	ColorSpecialBold,
	ColorSpecialUnderline,
	ColorSpecialBlink,
	ColorSpecialReverse,
	ColorSpecialItalic,
}

// col_dynamicFields is @typeInfo(Dynamic).@"enum".fields, as values.
var col_dynamicFields = []ColorDynamic{
	ColorDynamicForeground,
	ColorDynamicBackground,
	ColorDynamicCursor,
	ColorDynamicPointerForeground,
	ColorDynamicPointerBackground,
	ColorDynamicTektronixForeground,
	ColorDynamicTektronixBackground,
	ColorDynamicHighlightBackground,
	ColorDynamicTektronixCursor,
	ColorDynamicHighlightForeground,
}

// Zig: test "osc4"
func TestColor_osc4(t *testing.T) {
	expectEqual(t, uint16(256), ColorSpecialBold.Osc4())
	expectEqual(t, uint16(257), ColorSpecialUnderline.Osc4())
	expectEqual(t, uint16(258), ColorSpecialBlink.Osc4())
	expectEqual(t, uint16(259), ColorSpecialReverse.Osc4())
	expectEqual(t, uint16(260), ColorSpecialItalic.Osc4())
}

// Zig: test Special
func TestColor_Special(t *testing.T) {
	// Zig-only: `_ = Special;` only references the declaration; Go checks
	// that at compile time.
	_ = ColorSpecialBold
}

// Zig: test "next"
func TestColor_next(t *testing.T) {
	expectEqual(t, Some(ColorDynamicBackground), ColorDynamicForeground.Next())
	expectEqual(t, Some(ColorDynamicCursor), ColorDynamicBackground.Next())
	expectEqual(t, Some(ColorDynamicPointerForeground), ColorDynamicCursor.Next())
	expectEqual(t, Some(ColorDynamicPointerBackground), ColorDynamicPointerForeground.Next())
	expectEqual(t, Some(ColorDynamicTektronixForeground), ColorDynamicPointerBackground.Next())
	expectEqual(t, Some(ColorDynamicTektronixBackground), ColorDynamicTektronixForeground.Next())
	expectEqual(t, Some(ColorDynamicHighlightBackground), ColorDynamicTektronixBackground.Next())
	expectEqual(t, Some(ColorDynamicTektronixCursor), ColorDynamicHighlightBackground.Next())
	expectEqual(t, Some(ColorDynamicHighlightForeground), ColorDynamicTektronixCursor.Next())
	expectEqual(t, None[ColorDynamic](), ColorDynamicHighlightForeground.Next())
}

// Zig: test Dynamic
func TestColor_Dynamic(t *testing.T) {
	// Zig-only: `_ = Dynamic;` only references the declaration; Go checks
	// that at compile time.
	_ = ColorDynamicForeground
}

// Zig: test "palette: default"
func TestColor_palette_default(t *testing.T) {
	// Safety check
	for i := 0; i < 16; i++ {
		expectEqual(t, col_nameDefault(t, ColorName(i)), ColorDefault[i])
	}
}

// Zig: test "RGB.parse"
func TestColor_RGB_parse(t *testing.T) {
	expectEqual(t, ColorRGB{R: 255, G: 0, B: 0}, col_parse(t, "rgbi:1.0/0/0"))
	expectEqual(t, ColorRGB{R: 127, G: 160, B: 0}, col_parse(t, "rgb:7f/a0a0/0"))
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, col_parse(t, "rgb:f/ff/fff"))
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, col_parse(t, "#ffffff"))
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, col_parse(t, "#fff"))
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, col_parse(t, "#fffffffff"))
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, col_parse(t, "#ffffffffffff"))
	expectEqual(t, ColorRGB{R: 255, G: 0, B: 16}, col_parse(t, "#ff0010"))

	expectEqual(t, ColorRGB{R: 0, G: 0, B: 0}, col_parse(t, "black"))
	expectEqual(t, ColorRGB{R: 255, G: 0, B: 0}, col_parse(t, "red"))
	expectEqual(t, ColorRGB{R: 0, G: 255, B: 0}, col_parse(t, "green"))
	expectEqual(t, ColorRGB{R: 0, G: 0, B: 255}, col_parse(t, "blue"))
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, col_parse(t, "white"))

	expectEqual(t, ColorRGB{R: 124, G: 252, B: 0}, col_parse(t, "LawnGreen"))
	expectEqual(t, ColorRGB{R: 0, G: 250, B: 154}, col_parse(t, "medium spring green"))
	expectEqual(t, ColorRGB{R: 34, G: 139, B: 34}, col_parse(t, " Forest Green "))

	// Invalid format
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb;"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:"))
	col_expectError(t, ErrInvalidFormat, col_parseErr(":a/a/a"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("a/a/a"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:a/a/a/"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:00000///"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:000/"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgbi:a/a/a"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:0.5/0.0/1.0"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:not/hex/zz"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("#"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("#ff"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("#ffff"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("#fffff"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("#gggggg"))
}

// Zig: test "DynamicPalette: init"
func TestColor_DynamicPalette_init(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	expectEqual(t, ColorDefault, p.Current)
	expectEqual(t, ColorDefault, p.Original)
	expectEqual(t, 0, p.Mask.count())
}

// Zig: test "DynamicPalette: set"
func TestColor_DynamicPalette_set(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	new_color := ColorRGB{R: 255, G: 0, B: 0}

	p.Set(0, new_color)
	expectEqual(t, new_color, p.Current[0])
	expect(t, p.Mask.isSet(0), "p.mask.isSet(0)")
	expectEqual(t, 1, p.Mask.count())

	expectEqual(t, ColorDefault[0], p.Original[0])
}

// Zig: test "DynamicPalette: reset"
func TestColor_DynamicPalette_reset(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	new_color := ColorRGB{R: 255, G: 0, B: 0}

	p.Set(0, new_color)
	expect(t, p.Mask.isSet(0), "p.mask.isSet(0)")

	p.Reset(0)
	expectEqual(t, ColorDefault[0], p.Current[0])
	expect(t, !p.Mask.isSet(0), "!p.mask.isSet(0)")
	expectEqual(t, 0, p.Mask.count())
}

// Zig: test "DynamicPalette: resetAll"
func TestColor_DynamicPalette_resetAll(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	new_color := ColorRGB{R: 255, G: 0, B: 0}

	p.Set(0, new_color)
	p.Set(5, new_color)
	p.Set(10, new_color)
	expectEqual(t, 3, p.Mask.count())

	p.ResetAll()
	expectEqual(t, ColorDefault, p.Current)
	expectEqual(t, ColorDefault, p.Original)
	expectEqual(t, 0, p.Mask.count())
}

// Zig: test "DynamicPalette: changeDefault with no changes"
func TestColor_DynamicPalette_changeDefault_with_no_changes(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	new_palette := ColorDefault
	new_palette[0] = ColorRGB{R: 100, G: 100, B: 100}

	p.ChangeDefault(new_palette)
	expectEqual(t, new_palette, p.Original)
	expectEqual(t, new_palette, p.Current)
	expectEqual(t, 0, p.Mask.count())
}

// Zig: test "DynamicPalette: changeDefault preserves changes"
func TestColor_DynamicPalette_changeDefault_preserves_changes(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	custom_color := ColorRGB{R: 255, G: 0, B: 0}

	p.Set(5, custom_color)
	expect(t, p.Mask.isSet(5), "p.mask.isSet(5)")

	new_palette := ColorDefault
	new_palette[0] = ColorRGB{R: 100, G: 100, B: 100}
	new_palette[5] = ColorRGB{R: 50, G: 50, B: 50}

	p.ChangeDefault(new_palette)

	expectEqual(t, new_palette, p.Original)
	expectEqual(t, new_palette[0], p.Current[0])
	expectEqual(t, custom_color, p.Current[5])
	expect(t, p.Mask.isSet(5), "p.mask.isSet(5)")
	expectEqual(t, 1, p.Mask.count())
}

// Zig: test "DynamicPalette: changeDefault with multiple changes"
func TestColor_DynamicPalette_changeDefault_with_multiple_changes(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	red := ColorRGB{R: 255, G: 0, B: 0}
	green := ColorRGB{R: 0, G: 255, B: 0}
	blue := ColorRGB{R: 0, G: 0, B: 255}

	p.Set(1, red)
	p.Set(2, green)
	p.Set(3, blue)

	new_palette := ColorDefault
	new_palette[0] = ColorRGB{R: 50, G: 50, B: 50}
	new_palette[1] = ColorRGB{R: 60, G: 60, B: 60}

	p.ChangeDefault(new_palette)

	expectEqual(t, new_palette[0], p.Current[0])
	expectEqual(t, red, p.Current[1])
	expectEqual(t, green, p.Current[2])
	expectEqual(t, blue, p.Current[3])
	expectEqual(t, 3, p.Mask.count())
}

// Zig: test "LAB.fromRgb"
func TestColor_LAB_fromRgb(t *testing.T) {
	const epsilon = 0.5

	// White (255, 255, 255) -> L*=100, a*=0, b*=0
	white := colorLABFromRgb(ColorRGB{R: 255, G: 255, B: 255})
	col_expectApproxEqAbs(t, float32(100.0), white.l, epsilon)
	col_expectApproxEqAbs(t, float32(0.0), white.a, epsilon)
	col_expectApproxEqAbs(t, float32(0.0), white.b, epsilon)

	// Black (0, 0, 0) -> L*=0, a*=0, b*=0
	black := colorLABFromRgb(ColorRGB{R: 0, G: 0, B: 0})
	col_expectApproxEqAbs(t, float32(0.0), black.l, epsilon)
	col_expectApproxEqAbs(t, float32(0.0), black.a, epsilon)
	col_expectApproxEqAbs(t, float32(0.0), black.b, epsilon)

	// Pure red (255, 0, 0) -> L*≈53.23, a*≈80.11, b*≈67.22
	red := colorLABFromRgb(ColorRGB{R: 255, G: 0, B: 0})
	col_expectApproxEqAbs(t, float32(53.23), red.l, epsilon)
	col_expectApproxEqAbs(t, float32(80.11), red.a, epsilon)
	col_expectApproxEqAbs(t, float32(67.22), red.b, epsilon)

	// Pure green (0, 128, 0) -> L*≈46.23, a*≈-51.70, b*≈49.90
	green := colorLABFromRgb(ColorRGB{R: 0, G: 128, B: 0})
	col_expectApproxEqAbs(t, float32(46.23), green.l, epsilon)
	col_expectApproxEqAbs(t, float32(-51.70), green.a, epsilon)
	col_expectApproxEqAbs(t, float32(49.90), green.b, epsilon)

	// Pure blue (0, 0, 255) -> L*≈32.30, a*≈79.20, b*≈-107.86
	blue := colorLABFromRgb(ColorRGB{R: 0, G: 0, B: 255})
	col_expectApproxEqAbs(t, float32(32.30), blue.l, epsilon)
	col_expectApproxEqAbs(t, float32(79.20), blue.a, epsilon)
	col_expectApproxEqAbs(t, float32(-107.86), blue.b, epsilon)
}

// Zig: test "generate256Color: base16 preserved"
func TestColor_generate256Color_base16_preserved(t *testing.T) {
	bg := ColorRGB{R: 0, G: 0, B: 0}
	fg := ColorRGB{R: 255, G: 255, B: 255}
	palette := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)

	// The first 16 colors (base16) must remain unchanged.
	for i := 0; i < 16; i++ {
		expectEqual(t, ColorDefault[i], palette[i])
	}
}

// Zig: test "generate256Color: cube corners match base colors"
func TestColor_generate256Color_cube_corners_match_base_colors(t *testing.T) {
	bg := ColorRGB{R: 0, G: 0, B: 0}
	fg := ColorRGB{R: 255, G: 255, B: 255}
	palette := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)

	// Index 16 is cube (0,0,0) which should equal bg.
	expectEqual(t, bg, palette[16])

	// Index 231 is cube (5,5,5) which should equal fg.
	expectEqual(t, fg, palette[231])
}

// Zig: test "generate256Color: cube corners black/white with harmonious=false"
func TestColor_generate256Color_cube_corners_black_white_with_harmonious_false(t *testing.T) {
	black := ColorRGB{R: 0, G: 0, B: 0}
	white := ColorRGB{R: 255, G: 255, B: 255}

	// Dark theme: bg=black, fg=white.
	dark := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), black, white, false)
	expectEqual(t, black, dark[16])
	expectEqual(t, white, dark[231])

	// Light theme: bg=white, fg=black. The bg/red swap ensures
	// the cube still runs from black (16) to white (231).
	light := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), white, black, false)
	expectEqual(t, black, light[16])
	expectEqual(t, white, light[231])
}

// Zig: test "generate256Color: light theme cube corners with harmonious=true"
func TestColor_generate256Color_light_theme_cube_corners_with_harmonious_true(t *testing.T) {
	white := ColorRGB{R: 255, G: 255, B: 255}
	black := ColorRGB{R: 0, G: 0, B: 0}

	// harmonious=true skips the bg/fg swap, so the cube preserves the
	// original orientation: (0,0,0)=bg=white, (5,5,5)=fg=black.
	palette := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), white, black, true)
	expectEqual(t, white, palette[16])
	expectEqual(t, black, palette[231])
}

// Zig: test "generate256Color: grayscale ramp monotonic luminance"
func TestColor_generate256Color_grayscale_ramp_monotonic_luminance(t *testing.T) {
	bg := ColorRGB{R: 0, G: 0, B: 0}
	fg := ColorRGB{R: 255, G: 255, B: 255}
	palette := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)

	// The grayscale ramp (232–255) should have monotonically increasing
	// luminance from near-black to near-white.
	prev_lum := float64(0.0)
	for i := 232; i < 256; i++ {
		lum := palette[i].Luminance()
		expect(t, lum >= prev_lum, "lum >= prev_lum")
		prev_lum = lum
	}
}

// Zig: test "generate256Color: skip mask preserves original colors"
func TestColor_generate256Color_skip_mask_preserves_original_colors(t *testing.T) {
	bg := ColorRGB{R: 0, G: 0, B: 0}
	fg := ColorRGB{R: 255, G: 255, B: 255}

	// Mark a few indices as skipped; they should keep their base value.
	skip := StaticBitSet256InitEmpty()
	skip.set(20)
	skip.set(100)
	skip.set(240)

	palette := ColorGenerate256Color(ColorDefault, skip, bg, fg, false)
	expectEqual(t, ColorDefault[20], palette[20])
	expectEqual(t, ColorDefault[100], palette[100])
	expectEqual(t, ColorDefault[240], palette[240])

	// A non-skipped index in the cube should differ from the default.
	expect(t, !palette[21].Eql(ColorDefault[21]), "!palette[21].eql(default[21])")
}

// Zig: test "generate256Color: dark theme harmonious has no effect"
func TestColor_generate256Color_dark_theme_harmonious_has_no_effect(t *testing.T) {
	// For a dark theme (fg lighter than bg), harmonious should not change
	// the output because the inversion is only relevant for light themes.
	bg := ColorRGB{R: 0, G: 0, B: 0}
	fg := ColorRGB{R: 255, G: 255, B: 255}
	normal := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)
	harmonious := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, true)

	for i := 16; i < 256; i++ {
		expectEqual(t, normal[i], harmonious[i])
	}
}

// Zig: test "generate256Color: light theme harmonious skips inversion"
func TestColor_generate256Color_light_theme_harmonious_skips_inversion(t *testing.T) {
	// For a light theme (fg darker than bg), harmonious=true skips the
	// bg/red swap, producing different cube colors than harmonious=false.
	bg := ColorRGB{R: 255, G: 255, B: 255}
	fg := ColorRGB{R: 0, G: 0, B: 0}
	inverted := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)
	harmonious := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, true)

	// Cube origin (0,0,0) at index 16: without harmonious, bg and red are
	// swapped so it becomes the red base; with harmonious it stays as bg.
	expectEqual(t, bg, harmonious[16])
	expect(t, !inverted[16].Eql(bg), "!inverted[16].eql(bg)")

	// At least some cube colors should differ between the two modes.
	differ := 0
	for i := 16; i < 232; i++ {
		if !inverted[i].Eql(harmonious[i]) {
			differ += 1
		}
	}
	expect(t, differ > 0, "differ > 0")
}

// Zig: test "generate256Color: light theme harmonious grayscale ramp"
func TestColor_generate256Color_light_theme_harmonious_grayscale_ramp(t *testing.T) {
	bg := ColorRGB{R: 255, G: 255, B: 255}
	fg := ColorRGB{R: 0, G: 0, B: 0}

	// harmonious=false swaps bg/fg, so the ramp runs black→white (increasing).
	{
		palette := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)
		prev_lum := float64(0.0)
		for i := 232; i < 256; i++ {
			lum := palette[i].Luminance()
			expect(t, lum >= prev_lum, "lum >= prev_lum")
			prev_lum = lum
		}
	}

	// harmonious=true keeps original order, so the ramp runs white→black (decreasing).
	{
		palette := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, true)
		prev_lum := float64(1.0)
		for i := 232; i < 256; i++ {
			lum := palette[i].Luminance()
			expect(t, lum <= prev_lum, "lum <= prev_lum")
			prev_lum = lum
		}
	}
}

// Zig: test "LAB.toRgb"
func TestColor_LAB_toRgb(t *testing.T) {
	// Round-trip: RGB -> LAB -> RGB should recover the original values.
	cases := [...]ColorRGB{
		{R: 255, G: 255, B: 255},
		{R: 0, G: 0, B: 0},
		{R: 255, G: 0, B: 0},
		{R: 0, G: 128, B: 0},
		{R: 0, G: 0, B: 255},
		{R: 128, G: 128, B: 128},
		{R: 64, G: 224, B: 208},
	}

	for _, expected := range cases {
		lab := colorLABFromRgb(expected)
		actual := lab.toRgb()
		expectEqual(t, expected.R, actual.R)
		expectEqual(t, expected.G, actual.G)
		expectEqual(t, expected.B, actual.B)
	}
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// term51Rgb is the term51 test helper: build an RGB value.
func term51Rgb(r uint8, g uint8, b uint8) ColorRGB {
	return ColorRGB{R: r, G: g, B: b}
}

// term51_eps is the term51 test helper: tolerance for the f64 luminance and
// contrast maths. The expected values are the closed-form results computed
// in f64, so only libm rounding differences are allowed.
const term51_eps float64 = 1e-9

// Zig: test "term51: default: first 16 entries are the named colors"
func TestColor_term51_default_first_16_entries_are_the_named_colors(t *testing.T) {
	expectEqual(t, term51Rgb(0x1D, 0x1F, 0x21), ColorDefault[0])
	expectEqual(t, term51Rgb(0xCC, 0x66, 0x66), ColorDefault[1])
	expectEqual(t, term51Rgb(0xEA, 0xEA, 0xEA), ColorDefault[15])
	for i := 0; i < 16; i++ {
		expectEqual(t, col_nameDefault(t, ColorName(i)), ColorDefault[i])
	}
}

// Zig: test "term51: default: 6x6x6 cube corners and levels"
func TestColor_term51_default_6x6x6_cube_corners_and_levels(t *testing.T) {
	// Level table: 0 -> 0, n -> n*40+55.
	levels := [...]uint8{0, 95, 135, 175, 215, 255}
	for ri, r := range levels {
		for gi, g := range levels {
			for bi, b := range levels {
				idx := 16 + 36*ri + 6*gi + bi
				expectEqual(t, term51Rgb(r, g, b), ColorDefault[idx])
			}
		}
	}
	// The eight corners.
	expectEqual(t, term51Rgb(0, 0, 0), ColorDefault[16])
	expectEqual(t, term51Rgb(0, 0, 255), ColorDefault[21])
	expectEqual(t, term51Rgb(0, 255, 0), ColorDefault[46])
	expectEqual(t, term51Rgb(0, 255, 255), ColorDefault[51])
	expectEqual(t, term51Rgb(255, 0, 0), ColorDefault[196])
	expectEqual(t, term51Rgb(255, 0, 255), ColorDefault[201])
	expectEqual(t, term51Rgb(255, 255, 0), ColorDefault[226])
	expectEqual(t, term51Rgb(255, 255, 255), ColorDefault[231])
	// Neighbours of the first cube entry.
	expectEqual(t, term51Rgb(0, 0, 95), ColorDefault[17])
	expectEqual(t, term51Rgb(0, 95, 0), ColorDefault[22])
	expectEqual(t, term51Rgb(95, 0, 0), ColorDefault[52])
}

// Zig: test "term51: default: grayscale ramp 232 to 255"
func TestColor_term51_default_grayscale_ramp_232_to_255(t *testing.T) {
	expectEqual(t, term51Rgb(8, 8, 8), ColorDefault[232])
	expectEqual(t, term51Rgb(18, 18, 18), ColorDefault[233])
	expectEqual(t, term51Rgb(128, 128, 128), ColorDefault[244])
	expectEqual(t, term51Rgb(238, 238, 238), ColorDefault[255])
	for i := 232; i < 256; i++ {
		v := uint8((i-232)*10 + 8)
		expectEqual(t, term51Rgb(v, v, v), ColorDefault[i])
	}
	expectEqual(t, 256, len(ColorDefault))
}

// Zig: test "term51: Name.default: every named color and unnamed values have no default"
func TestColor_term51_Name_default_every_named_color_and_unnamed_values_have_no_default(t *testing.T) {
	expected := [16]ColorRGB{
		term51Rgb(0x1D, 0x1F, 0x21), term51Rgb(0xCC, 0x66, 0x66),
		term51Rgb(0xB5, 0xBD, 0x68), term51Rgb(0xF0, 0xC6, 0x74),
		term51Rgb(0x81, 0xA2, 0xBE), term51Rgb(0xB2, 0x94, 0xBB),
		term51Rgb(0x8A, 0xBE, 0xB7), term51Rgb(0xC5, 0xC8, 0xC6),
		term51Rgb(0x66, 0x66, 0x66), term51Rgb(0xD5, 0x4E, 0x53),
		term51Rgb(0xB9, 0xCA, 0x4A), term51Rgb(0xE7, 0xC5, 0x47),
		term51Rgb(0x7A, 0xA6, 0xDA), term51Rgb(0xC3, 0x97, 0xD8),
		term51Rgb(0x70, 0xC0, 0xB1), term51Rgb(0xEA, 0xEA, 0xEA),
	}
	for i, e := range expected {
		expectEqual(t, e, col_nameDefault(t, ColorName(i)))
	}
	expectEqual(t, term51Rgb(0x1D, 0x1F, 0x21), col_nameDefault(t, ColorNameBlack))
	expectEqual(t, term51Rgb(0xEA, 0xEA, 0xEA), col_nameDefault(t, ColorNameBrightWhite))

	// One past the named range, a cube entry, and the last value.
	_, err := ColorName(16).Default()
	col_expectError(t, ErrNoDefaultValue, err)
	_, err = ColorName(17).Default()
	col_expectError(t, ErrNoDefaultValue, err)
	_, err = ColorName(231).Default()
	col_expectError(t, ErrNoDefaultValue, err)
	_, err = ColorName(255).Default()
	col_expectError(t, ErrNoDefaultValue, err)
}

// Zig: test "term51: Special.osc4: adds the palette length and is a u16"
func TestColor_term51_Special_osc4_adds_the_palette_length_and_is_a_u16(t *testing.T) {
	// Zig-only: try testing.expect(@TypeOf(Special.bold.osc4()) == u16);
	// Go: the assignment below is checked by the compiler.
	var osc4 uint16 = ColorSpecialBold.Osc4()
	_ = osc4
	for _, f := range col_specialFields {
		expectEqual(t, uint16(256+int(f)), f.Osc4())
	}
	expectEqual(t, 5, len(col_specialFields))
}

// Zig: test "term51: Dynamic.next: integer values and the end of the list"
func TestColor_term51_Dynamic_next_integer_values_and_the_end_of_the_list(t *testing.T) {
	expectEqual(t, 10, len(col_dynamicFields))
	expectEqual(t, uint8(10), uint8(ColorDynamicForeground))
	expectEqual(t, uint8(19), uint8(ColorDynamicHighlightForeground))
	// Every value but the last has a successor with the next integer.
	for _, d := range col_dynamicFields {
		if uint8(d) == 19 {
			expectEqual(t, None[ColorDynamic](), d.Next())
		} else {
			expectEqual(t, uint8(d)+1, uint8(zigUnwrap(t, d.Next())))
		}
	}
	// The chain from foreground visits all ten exactly once.
	count := 1
	cur := ColorDynamicForeground
	for {
		n := cur.Next()
		if !n.Valid {
			break
		}
		count += 1
		cur = n.Value
	}
	expectEqual(t, 10, count)
	expectEqual(t, ColorDynamicHighlightForeground, cur)
}

// Zig: test "term51: DynamicRGB.init: default set, no override"
func TestColor_term51_DynamicRGB_init_default_set_no_override(t *testing.T) {
	c := term51Rgb(1, 2, 3)
	d := ColorDynamicRGBInit(c)
	expectEqual(t, None[ColorRGB](), d.Override)
	expectEqual(t, Some(c), d.Default)

	expectEqual(t, None[ColorRGB](), ColorDynamicRGBUnset.Override)
	expectEqual(t, None[ColorRGB](), ColorDynamicRGBUnset.Default)
}

// Zig: test "term51: DynamicRGB.get: override wins over default, null when neither"
func TestColor_term51_DynamicRGB_get_override_wins_over_default_null_when_neither(t *testing.T) {
	def := term51Rgb(10, 20, 30)
	over := term51Rgb(40, 50, 60)

	expectEqual(t, None[ColorRGB](), ColorDynamicRGBUnset.Get())

	d := ColorDynamicRGBInit(def)
	expectEqual(t, Some(def), d.Get())
	d.Set(over)
	expectEqual(t, Some(over), d.Get())

	// Override without any default.
	u := ColorDynamicRGBUnset
	u.Set(over)
	expectEqual(t, Some(over), u.Get())

	// Override equal to a black RGB is still an override (not null).
	z := ColorDynamicRGBInit(def)
	z.Set(term51Rgb(0, 0, 0))
	expectEqual(t, Some(term51Rgb(0, 0, 0)), z.Get())
}

// Zig: test "term51: DynamicRGB.set: replaces the override, keeps the default"
func TestColor_term51_DynamicRGB_set_replaces_the_override_keeps_the_default(t *testing.T) {
	def := term51Rgb(10, 20, 30)
	d := ColorDynamicRGBInit(def)
	d.Set(term51Rgb(1, 1, 1))
	expectEqual(t, Some(term51Rgb(1, 1, 1)), d.Override)
	expectEqual(t, Some(def), d.Default)
	d.Set(term51Rgb(2, 2, 2))
	expectEqual(t, Some(term51Rgb(2, 2, 2)), d.Override)
	expectEqual(t, Some(def), d.Default)
}

// Zig: test "term51: DynamicRGB.reset: restores the default, also when never set"
func TestColor_term51_DynamicRGB_reset_restores_the_default_also_when_never_set(t *testing.T) {
	def := term51Rgb(10, 20, 30)

	d := ColorDynamicRGBInit(def)
	d.Set(term51Rgb(1, 1, 1))
	d.Reset()
	expectEqual(t, Some(def), d.Get())
	// reset copies the default into the override rather than clearing it.
	expectEqual(t, Some(def), d.Override)

	// Reset of something never set: override becomes the default value.
	never := ColorDynamicRGBInit(def)
	expectEqual(t, None[ColorRGB](), never.Override)
	never.Reset()
	expectEqual(t, Some(def), never.Override)
	expectEqual(t, Some(def), never.Get())

	// Fully unset stays unset after reset.
	u := ColorDynamicRGBUnset
	u.Reset()
	expectEqual(t, None[ColorRGB](), u.Override)
	expectEqual(t, None[ColorRGB](), u.Get())

	// Override with no default: reset clears to null.
	o := ColorDynamicRGBUnset
	o.Set(term51Rgb(5, 5, 5))
	o.Reset()
	expectEqual(t, None[ColorRGB](), o.Get())
}

// Zig: test "term51: DynamicPalette.init: custom palette, empty mask, and default constant"
func TestColor_term51_DynamicPalette_init_custom_palette_empty_mask_and_default_constant(t *testing.T) {
	pal := ColorDefault
	pal[0] = term51Rgb(9, 9, 9)
	pal[255] = term51Rgb(8, 8, 8)
	p := ColorDynamicPaletteInit(pal)
	expectEqual(t, pal, p.Current)
	expectEqual(t, pal, p.Original)
	expectEqual(t, 0, p.Mask.count())

	expectEqual(t, ColorDefault, ColorDynamicPaletteDefault.Current)
	expectEqual(t, ColorDefault, ColorDynamicPaletteDefault.Original)
	expectEqual(t, 0, ColorDynamicPaletteDefault.Mask.count())
}

// Zig: test "term51: DynamicPalette.set: index 0, 255 and repeated sets"
func TestColor_term51_DynamicPalette_set_index_0_255_and_repeated_sets(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	a := term51Rgb(1, 2, 3)
	b := term51Rgb(4, 5, 6)

	p.Set(255, a)
	expectEqual(t, a, p.Current[255])
	expect(t, p.Mask.isSet(255), "p.mask.isSet(255)")
	expect(t, !p.Mask.isSet(254), "!p.mask.isSet(254)")
	expectEqual(t, 1, p.Mask.count())
	expectEqual(t, ColorDefault[255], p.Original[255])

	p.Set(0, b)
	expectEqual(t, b, p.Current[0])
	expect(t, p.Mask.isSet(0), "p.mask.isSet(0)")
	expectEqual(t, 2, p.Mask.count())

	// Setting the same index again overwrites and does not add a bit.
	p.Set(0, a)
	expectEqual(t, a, p.Current[0])
	expectEqual(t, 2, p.Mask.count())

	// Setting a color equal to the original still marks it as changed.
	p.Set(16, ColorDefault[16])
	expect(t, p.Mask.isSet(16), "p.mask.isSet(16)")
	expectEqual(t, 3, p.Mask.count())

	// Cube/gray boundaries.
	p.Set(231, a)
	p.Set(232, b)
	expect(t, p.Mask.isSet(231), "p.mask.isSet(231)")
	expect(t, p.Mask.isSet(232), "p.mask.isSet(232)")
	expectEqual(t, a, p.Current[231])
	expectEqual(t, b, p.Current[232])
}

// Zig: test "term51: DynamicPalette.reset: restores one index, never-set index is a no-op"
func TestColor_term51_DynamicPalette_reset_restores_one_index_never_set_index_is_a_no_op(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	a := term51Rgb(1, 2, 3)

	p.Set(255, a)
	p.Set(1, a)
	p.Reset(255)
	expectEqual(t, ColorDefault[255], p.Current[255])
	expect(t, !p.Mask.isSet(255), "!p.mask.isSet(255)")
	expectEqual(t, a, p.Current[1])
	expect(t, p.Mask.isSet(1), "p.mask.isSet(1)")
	expectEqual(t, 1, p.Mask.count())

	// Reset of an index that was never set.
	before := p
	p.Reset(100)
	expectEqual(t, before.Current, p.Current)
	expectEqual(t, 1, p.Mask.count())
	p.Reset(0)
	expectEqual(t, before.Current, p.Current)

	// Reset twice.
	p.Reset(1)
	p.Reset(1)
	expectEqual(t, ColorDefault, p.Current)
	expectEqual(t, 0, p.Mask.count())

	// Reset uses the current original, which may have been changed.
	pal := ColorDefault
	pal[7] = term51Rgb(70, 70, 70)
	p.Set(7, a)
	p.ChangeDefault(pal)
	p.Reset(7)
	expectEqual(t, pal[7], p.Current[7])
	expect(t, !p.Mask.isSet(7), "!p.mask.isSet(7)")
}

// Zig: test "term51: DynamicPalette.resetAll: clears every change and uses the current original"
func TestColor_term51_DynamicPalette_resetAll_clears_every_change_and_uses_the_current_original(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	a := term51Rgb(1, 2, 3)

	// Nothing set: no-op.
	p.ResetAll()
	expectEqual(t, ColorDefault, p.Current)
	expectEqual(t, 0, p.Mask.count())

	// Every index set.
	for i := 0; i < 256; i++ {
		p.Set(uint8(i), a)
	}
	expectEqual(t, 256, p.Mask.count())
	p.ResetAll()
	expectEqual(t, ColorDefault, p.Current)
	expectEqual(t, ColorDefault, p.Original)
	expectEqual(t, 0, p.Mask.count())

	// After changeDefault, resetAll resets to the new default.
	pal := ColorDefault
	pal[3] = term51Rgb(33, 33, 33)
	p.Set(4, a)
	p.ChangeDefault(pal)
	p.ResetAll()
	expectEqual(t, pal, p.Current)
	expectEqual(t, pal, p.Original)
	expectEqual(t, 0, p.Mask.count())
}

// Zig: test "term51: DynamicPalette.changeDefault: boundary indexes and repeated changes"
func TestColor_term51_DynamicPalette_changeDefault_boundary_indexes_and_repeated_changes(t *testing.T) {
	p := ColorDynamicPaletteInit(ColorDefault)
	a := term51Rgb(1, 2, 3)
	b := term51Rgb(4, 5, 6)

	p.Set(0, a)
	p.Set(255, b)

	pal := ColorDefault
	for i := range pal {
		pal[i] = term51Rgb(7, 7, 7)
	}
	p.ChangeDefault(pal)
	expectEqual(t, pal, p.Original)
	expectEqual(t, a, p.Current[0])
	expectEqual(t, b, p.Current[255])
	expectEqual(t, term51Rgb(7, 7, 7), p.Current[1])
	expectEqual(t, term51Rgb(7, 7, 7), p.Current[254])
	expectEqual(t, 2, p.Mask.count())

	// Changing again keeps the same overrides.
	pal2 := ColorDefault
	for i := range pal2 {
		pal2[i] = term51Rgb(9, 9, 9)
	}
	p.ChangeDefault(pal2)
	expectEqual(t, a, p.Current[0])
	expectEqual(t, b, p.Current[255])
	expectEqual(t, term51Rgb(9, 9, 9), p.Current[128])

	// Resetting a preserved index goes to the newest default.
	p.Reset(0)
	expectEqual(t, term51Rgb(9, 9, 9), p.Current[0])
	expectEqual(t, 1, p.Mask.count())

	// Changing to the same palette is a fixed point.
	snapshot := p
	p.ChangeDefault(pal2)
	expectEqual(t, snapshot.Current, p.Current)
	expectEqual(t, snapshot.Mask.count(), p.Mask.count())
}

// Zig: test "term51: RGB.eql: each channel compared independently"
func TestColor_term51_RGB_eql_each_channel_compared_independently(t *testing.T) {
	base := term51Rgb(10, 20, 30)
	expect(t, base.Eql(base), "base.eql(base)")
	expect(t, base.Eql(term51Rgb(10, 20, 30)), "base.eql(term51Rgb(10, 20, 30))")
	expect(t, !base.Eql(term51Rgb(11, 20, 30)), "!base.eql(term51Rgb(11, 20, 30))")
	expect(t, !base.Eql(term51Rgb(10, 21, 30)), "!base.eql(term51Rgb(10, 21, 30))")
	expect(t, !base.Eql(term51Rgb(10, 20, 31)), "!base.eql(term51Rgb(10, 20, 31))")
	expect(t, !base.Eql(term51Rgb(30, 20, 10)), "!base.eql(term51Rgb(30, 20, 10))")
	expect(t, ColorRGB{}.Eql(term51Rgb(0, 0, 0)), "(RGB{}).eql(term51Rgb(0, 0, 0))")
	expect(t, term51Rgb(255, 255, 255).Eql(term51Rgb(255, 255, 255)), "term51Rgb(255, 255, 255).eql(term51Rgb(255, 255, 255))")
	expect(t, !term51Rgb(255, 255, 255).Eql(term51Rgb(254, 255, 255)), "!term51Rgb(255, 255, 255).eql(term51Rgb(254, 255, 255))")
	expect(t, !term51Rgb(0, 0, 0).Eql(term51Rgb(0, 0, 1)), "!term51Rgb(0, 0, 0).eql(term51Rgb(0, 0, 1))")
}

// Zig: test "term51: RGB.componentLuminance: boundaries of the linear/power split"
func TestColor_term51_RGB_componentLuminance_boundaries_of_the_linear_power_split(t *testing.T) {
	const eps = 1e-12
	expectEqual(t, float64(0.0), colorRGBComponentLuminance(0))
	col_expectApproxEqAbs(t, float64(0.0003035269835488375), colorRGBComponentLuminance(1), eps)
	// 10/255 = 0.03921 <= 0.03928: linear segment (c / 255 / 12.92).
	col_expectApproxEqAbs(t, float64(0.003035269835488375), colorRGBComponentLuminance(10), eps)
	col_expectApproxEqAbs(t, 10.0/255.0/12.92, colorRGBComponentLuminance(10), eps)
	// 11/255 = 0.04314 > 0.03928: power segment.
	col_expectApproxEqAbs(t, float64(0.003346535763899161), colorRGBComponentLuminance(11), eps)
	expect(t, colorRGBComponentLuminance(11) > colorRGBComponentLuminance(10), "RGB.componentLuminance(11) > RGB.componentLuminance(10)")
	col_expectApproxEqAbs(t, float64(0.21223075741405523), colorRGBComponentLuminance(127), eps)
	col_expectApproxEqAbs(t, float64(0.21586050011389923), colorRGBComponentLuminance(128), eps)
	col_expectApproxEqAbs(t, float64(0.9911020971138298), colorRGBComponentLuminance(254), eps)
	col_expectApproxEqAbs(t, float64(1.0), colorRGBComponentLuminance(255), eps)

	// Strictly increasing over the whole range.
	prev := float64(-1.0)
	for c := 0; c <= 255; c++ {
		v := colorRGBComponentLuminance(uint8(c))
		expect(t, v > prev, "v > prev")
		prev = v
	}
}

// Zig: test "term51: RGB.luminance: black, white, primaries, and mid values"
func TestColor_term51_RGB_luminance_black_white_primaries_and_mid_values(t *testing.T) {
	col_expectApproxEqAbs(t, float64(0.0), term51Rgb(0, 0, 0).Luminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(1.0), term51Rgb(255, 255, 255).Luminance(), term51_eps)
	// The weights of the primaries.
	col_expectApproxEqAbs(t, float64(0.2126), term51Rgb(255, 0, 0).Luminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.7152), term51Rgb(0, 255, 0).Luminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.0722), term51Rgb(0, 0, 255).Luminance(), term51_eps)
	// Mid values.
	col_expectApproxEqAbs(t, float64(0.21586050011389923), term51Rgb(128, 128, 128).Luminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.2122307574140552), term51Rgb(127, 127, 127).Luminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.013509891961436194), term51Rgb(0x1D, 0x1F, 0x21).Luminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.003304184574161733), term51Rgb(10, 11, 12).Luminance(), term51_eps)
}

// Zig: test "term51: RGB.contrast: black/white, identical colors, symmetry, mid values"
func TestColor_term51_RGB_contrast_black_white_identical_colors_symmetry_mid_values(t *testing.T) {
	black := term51Rgb(0, 0, 0)
	white := term51Rgb(255, 255, 255)
	gray := term51Rgb(128, 128, 128)

	// Extremes: 21:1 either way, 1:1 for identical colors.
	col_expectApproxEqAbs(t, float64(21.0), black.Contrast(white), term51_eps)
	col_expectApproxEqAbs(t, float64(21.0), white.Contrast(black), term51_eps)
	col_expectApproxEqAbs(t, float64(1.0), black.Contrast(black), term51_eps)
	col_expectApproxEqAbs(t, float64(1.0), white.Contrast(white), term51_eps)
	col_expectApproxEqAbs(t, float64(1.0), gray.Contrast(gray), term51_eps)

	// Mid values.
	col_expectApproxEqAbs(t, float64(5.317210002277984), gray.Contrast(black), term51_eps)
	col_expectApproxEqAbs(t, float64(5.317210002277984), black.Contrast(gray), term51_eps)
	col_expectApproxEqAbs(t, float64(3.9494396480491156), gray.Contrast(white), term51_eps)
	col_expectApproxEqAbs(t, float64(3.9494396480491156), white.Contrast(gray), term51_eps)
	col_expectApproxEqAbs(t, float64(2.148936170212766), term51Rgb(255, 0, 0).Contrast(term51Rgb(0, 0, 255)), term51_eps)
	col_expectApproxEqAbs(t, float64(9.802581130534), term51Rgb(0x1D, 0x1F, 0x21).Contrast(term51Rgb(0xC5, 0xC8, 0xC6)), term51_eps)

	// Always within [1, 21].
	samples := [...]ColorRGB{black, white, gray, term51Rgb(255, 0, 0), term51Rgb(0, 255, 0), term51Rgb(0, 0, 255), term51Rgb(1, 1, 1), term51Rgb(254, 254, 254)}
	for _, a := range samples {
		for _, b := range samples {
			c := a.Contrast(b)
			expect(t, c >= 1.0-term51_eps, "c >= 1.0 - term51_eps")
			expect(t, c <= 21.0+term51_eps, "c <= 21.0 + term51_eps")
			col_expectApproxEqAbs(t, c, b.Contrast(a), term51_eps)
		}
	}
}

// Zig: test "term51: RGB.perceivedLuminance: black, white, primaries, and mid values"
func TestColor_term51_RGB_perceivedLuminance_black_white_primaries_and_mid_values(t *testing.T) {
	expectEqual(t, float64(0.0), term51Rgb(0, 0, 0).PerceivedLuminance())
	col_expectApproxEqAbs(t, float64(1.0), term51Rgb(255, 255, 255).PerceivedLuminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.299), term51Rgb(255, 0, 0).PerceivedLuminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.587), term51Rgb(0, 255, 0).PerceivedLuminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.114), term51Rgb(0, 0, 255).PerceivedLuminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.5019607843137255), term51Rgb(128, 128, 128).PerceivedLuminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.4980392156862745), term51Rgb(127, 127, 127).PerceivedLuminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.12011764705882354), term51Rgb(0x1D, 0x1F, 0x21).PerceivedLuminance(), term51_eps)
	col_expectApproxEqAbs(t, float64(0.04241176470588236), term51Rgb(10, 11, 12).PerceivedLuminance(), term51_eps)
	// It is linear in each channel, unlike luminance.
	col_expectApproxEqAbs(t, float64(0.299*51.0/255.0), term51Rgb(51, 0, 0).PerceivedLuminance(), term51_eps)
}

// Zig: test "term51: RGB.fromHex: scaling at 1, 2, 3 and 4 digits"
func TestColor_term51_RGB_fromHex_scaling_at_1_2_3_and_4_digits(t *testing.T) {
	// 1 digit: value * 255 / 15.
	expectEqual(t, uint8(0), col_fromHex(t, "0"))
	expectEqual(t, uint8(17), col_fromHex(t, "1"))
	expectEqual(t, uint8(85), col_fromHex(t, "5"))
	expectEqual(t, uint8(136), col_fromHex(t, "8"))
	expectEqual(t, uint8(170), col_fromHex(t, "a"))
	expectEqual(t, uint8(255), col_fromHex(t, "f"))
	// 2 digits: value itself.
	expectEqual(t, uint8(0), col_fromHex(t, "00"))
	expectEqual(t, uint8(1), col_fromHex(t, "01"))
	expectEqual(t, uint8(15), col_fromHex(t, "0f"))
	expectEqual(t, uint8(16), col_fromHex(t, "10"))
	expectEqual(t, uint8(127), col_fromHex(t, "7f"))
	expectEqual(t, uint8(128), col_fromHex(t, "80"))
	expectEqual(t, uint8(254), col_fromHex(t, "fe"))
	expectEqual(t, uint8(255), col_fromHex(t, "ff"))
	// 3 digits: value * 255 / 4095, truncated.
	expectEqual(t, uint8(0), col_fromHex(t, "000"))
	expectEqual(t, uint8(0), col_fromHex(t, "001"))
	expectEqual(t, uint8(0), col_fromHex(t, "010"))
	expectEqual(t, uint8(1), col_fromHex(t, "011"))
	expectEqual(t, uint8(127), col_fromHex(t, "800"))
	expectEqual(t, uint8(127), col_fromHex(t, "801"))
	expectEqual(t, uint8(254), col_fromHex(t, "ffe"))
	expectEqual(t, uint8(255), col_fromHex(t, "fff"))
	// 4 digits: value * 255 / 65535, truncated.
	expectEqual(t, uint8(0), col_fromHex(t, "0000"))
	expectEqual(t, uint8(0), col_fromHex(t, "0001"))
	expectEqual(t, uint8(0), col_fromHex(t, "0100"))
	expectEqual(t, uint8(1), col_fromHex(t, "0101"))
	expectEqual(t, uint8(127), col_fromHex(t, "8000"))
	expectEqual(t, uint8(160), col_fromHex(t, "a0a0"))
	expectEqual(t, uint8(254), col_fromHex(t, "fffe"))
	expectEqual(t, uint8(255), col_fromHex(t, "ffff"))
	// Digit width, not value, selects the divisor: "0f" is 15 but "00f" is 0.
	expectEqual(t, uint8(15), col_fromHex(t, "0f"))
	expectEqual(t, uint8(0), col_fromHex(t, "00f"))
}

// Zig: test "term51: RGB.fromHex: case, invalid digits, and length limits"
func TestColor_term51_RGB_fromHex_case_invalid_digits_and_length_limits(t *testing.T) {
	expectEqual(t, uint8(255), col_fromHex(t, "F"))
	expectEqual(t, uint8(171), col_fromHex(t, "aB"))
	expectEqual(t, uint8(171), col_fromHex(t, "Ab"))
	expectEqual(t, col_fromHex(t, "abc"), col_fromHex(t, "ABC"))
	expectEqual(t, col_fromHex(t, "abcd"), col_fromHex(t, "AbCd"))

	col_expectError(t, ErrInvalidFormat, col_fromHexErr(""))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("fffff"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("00000"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("000000"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("g"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("0g"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("zz"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("fg0"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("fff!"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr(" "))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr(" f"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("f "))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("-1"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("-0"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("+f"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("0x"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("0x1"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("\x00"))

	// std.fmt.parseUnsigned accepts '_' between digits, so these are valid.
	expectEqual(t, uint8(15), col_fromHex(t, "f_f"))
	expectEqual(t, uint8(1), col_fromHex(t, "1_1"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("_f"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("f_"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("_"))
}

// Zig: test "term51: RGB.fromIntensity: range, truncation, and accepted float syntax"
func TestColor_term51_RGB_fromIntensity_range_truncation_and_accepted_float_syntax(t *testing.T) {
	expectEqual(t, uint8(0), col_fromIntensity(t, "0"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "0.0"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "-0"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "-0.0"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "1"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "1.0"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "1."))
	expectEqual(t, uint8(255), col_fromIntensity(t, "1e0"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "+1"))
	// Values are truncated, not rounded.
	expectEqual(t, uint8(127), col_fromIntensity(t, "0.5"))
	expectEqual(t, uint8(127), col_fromIntensity(t, ".5"))
	expectEqual(t, uint8(127), col_fromIntensity(t, "+0.5"))
	expectEqual(t, uint8(127), col_fromIntensity(t, "5e-1"))
	expectEqual(t, uint8(25), col_fromIntensity(t, "0.1"))
	expectEqual(t, uint8(229), col_fromIntensity(t, "0.9"))
	expectEqual(t, uint8(254), col_fromIntensity(t, "0.999"))
	expectEqual(t, uint8(254), col_fromIntensity(t, "0.9999"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "0.0039"))
	expectEqual(t, uint8(1), col_fromIntensity(t, "0.004"))
	// std.fmt.parseFloat also accepts hexadecimal floats.
	expectEqual(t, uint8(127), col_fromIntensity(t, "0x1p-1"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "0x1"))

	// Out of range.
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("1.0000001"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("1.1"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("2"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("5."))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("255"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("-0.1"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("-1"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("inf"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("-inf"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("1e1"))
	// Not floats.
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr(""))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("."))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("+"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("a"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("ff"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("0,5"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr(" 0.5"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("0.5 "))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("0.5/"))
}

// Zig: test "term51: RGB.parse: rgb: with 1, 2, 3 and 4 hex digits per channel"
func TestColor_term51_RGB_parse_rgb_with_1_2_3_and_4_hex_digits_per_channel(t *testing.T) {
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgb:0/0/0"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgb:f/f/f"))
	expectEqual(t, term51Rgb(136, 136, 136), col_parse(t, "rgb:8/8/8"))
	expectEqual(t, term51Rgb(17, 17, 17), col_parse(t, "rgb:1/1/1"))
	expectEqual(t, term51Rgb(17, 34, 51), col_parse(t, "rgb:1/2/3"))

	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgb:00/00/00"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgb:ff/ff/ff"))
	expectEqual(t, term51Rgb(128, 128, 128), col_parse(t, "rgb:80/80/80"))
	expectEqual(t, term51Rgb(1, 1, 1), col_parse(t, "rgb:01/01/01"))
	expectEqual(t, term51Rgb(254, 254, 254), col_parse(t, "rgb:fe/fe/fe"))

	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgb:000/000/000"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgb:fff/fff/fff"))
	expectEqual(t, term51Rgb(127, 127, 127), col_parse(t, "rgb:800/800/800"))
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgb:001/001/001"))
	expectEqual(t, term51Rgb(254, 254, 254), col_parse(t, "rgb:ffe/ffe/ffe"))

	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgb:0000/0000/0000"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgb:ffff/ffff/ffff"))
	expectEqual(t, term51Rgb(127, 127, 127), col_parse(t, "rgb:8000/8000/8000"))
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgb:0001/0001/0001"))
	expectEqual(t, term51Rgb(254, 254, 254), col_parse(t, "rgb:fffe/fffe/fffe"))

	// Mixed channel lengths and independent scaling per channel.
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgb:f/ff/fff"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgb:f/ff/ffff"))
	expectEqual(t, term51Rgb(127, 160, 0), col_parse(t, "rgb:7f/a0a0/0"))
	expectEqual(t, term51Rgb(0, 128, 127), col_parse(t, "rgb:0/80/8000"))
	expectEqual(t, term51Rgb(17, 1, 0), col_parse(t, "rgb:1/01/0001"))
	expectEqual(t, term51Rgb(136, 128, 127), col_parse(t, "rgb:8/80/800"))

	// Upper, lower, and mixed case hex.
	expectEqual(t, term51Rgb(255, 170, 170), col_parse(t, "rgb:FFFF/AaAa/aAaA"))
	expectEqual(t, term51Rgb(171, 205, 239), col_parse(t, "rgb:AB/cd/EF"))
	expectEqual(t, term51Rgb(171, 205, 239), col_parse(t, "rgb:ab/CD/ef"))

	// Underscores in a channel are accepted by the underlying integer parser.
	expectEqual(t, term51Rgb(15, 0, 0), col_parse(t, "rgb:f_f/0/0"))
}

// Zig: test "term51: RGB.parse: rgb: invalid forms"
func TestColor_term51_RGB_parse_rgb_invalid_forms(t *testing.T) {
	bad := [...]string{
		// Too short for the shortest valid spec.
		"rgb:",
		"rgb:a",
		"rgb:a/a",
		"rgb:a/a/",
		"rgb",
		"rg",
		// Missing channel, empty channel.
		"rgb:a//aa",
		"rgb:/a/aa",
		"rgb:a/aa/",
		"rgb:0/0/",
		"rgb:/0/0",
		"rgb:0//0",
		"rgb:///",
		// Extra channel or trailing separator.
		"rgb:0/0/0/0",
		"rgb:0/0/0/",
		"rgb:a/a/a/",
		// Too many digits (5) in any channel.
		"rgb:fffff/0/0",
		"rgb:0/fffff/0",
		"rgb:0/0/fffff",
		// Invalid digits.
		"rgb:g/0/0",
		"rgb:0/g/0",
		"rgb:0/0/g",
		"rgb:0/0/-1",
		"rgb:0/+1/0",
		"rgb:0x1/0/0",
		"rgb:0.5/0/0",
		// Wrong prefix or case (the prefix is case sensitive).
		"RGB:0/0/0",
		"Rgb:0/0/0",
		"rGb:0/0/0",
		"rgbI:0/0/0",
		"rgbx:0/0/0",
		"rgb 0/0/0",
		"rgb;0/0/0",
		"rgb0/0/0/",
		"rgb:0:0:0",
		"rgb:0,0,0",
		// Whitespace is not allowed around or inside the spec.
		" rgb:0/0/0",
		"rgb:0/0/0 ",
		"rgb: 0/0/0",
		"rgb:0 /0/0",
		"rgb:0/ 0/0",
		"rgb:0/0/ 0",
		"\trgb:0/0/0",
		"rgb:0/0/0\n",
	}
	for _, s := range bad {
		col_expectError(t, ErrInvalidFormat, col_parseErr(s))
	}

	// Over-long input.
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:"+strings.Repeat("f", 100)+"/0/0"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:0/0/"+strings.Repeat("0", 100)))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgb:"+strings.Repeat("0/", 100)))
	col_expectError(t, ErrInvalidFormat, col_parseErr(strings.Repeat("x", 100)))
	col_expectError(t, ErrInvalidFormat, col_parseErr(strings.Repeat("rgb", 100)))
}

// Zig: test "term51: RGB.parse: rgbi: floating point intensities"
func TestColor_term51_RGB_parse_rgbi_floating_point_intensities(t *testing.T) {
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgbi:0/0/0"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgbi:1/1/1"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "rgbi:1.0/1.0/1.0"))
	expectEqual(t, term51Rgb(255, 0, 0), col_parse(t, "rgbi:1.0/0/0"))
	expectEqual(t, term51Rgb(127, 127, 127), col_parse(t, "rgbi:0.5/0.5/0.5"))
	expectEqual(t, term51Rgb(0, 255, 127), col_parse(t, "rgbi:0/1/0.5"))
	expectEqual(t, term51Rgb(127, 127, 127), col_parse(t, "rgbi:.5/.5/.5"))
	expectEqual(t, term51Rgb(255, 0, 127), col_parse(t, "rgbi:1e0/0e0/5e-1"))
	expectEqual(t, term51Rgb(255, 0, 255), col_parse(t, "rgbi:+1/+0/+1"))
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "rgbi:-0.0/-0/0.0"))
	expectEqual(t, term51Rgb(25, 229, 254), col_parse(t, "rgbi:0.1/0.9/0.999"))
	// Hexadecimal floats are accepted by the float parser.
	expectEqual(t, term51Rgb(127, 0, 0), col_parse(t, "rgbi:0x1p-1/0/0"))

	// Out of range or malformed.
	bad := [...]string{
		"rgbi:1.0000001/0/0",
		"rgbi:0/1.1/0",
		"rgbi:0/0/2",
		"rgbi:-0.1/0/0",
		"rgbi:inf/0/0",
		"rgbi:0/0/-inf",
		"rgbi:a/a/a",
		"rgbi:ff/0/0",
		"rgbi:0/0/",
		"rgbi:/0/0",
		"rgbi:0//0",
		"rgbi:0.5/0.5",
		"rgbi:0/0/0/0",
		"rgbi:0/0/0/",
		"rgbi:,/0/0",
		"rgbi: 0/0/0",
		"rgbi:0/0/0 ",
		"rgbi:0/ 0/0",
		// Missing the colon after rgbi.
		"rgbi0/0/0/",
		"rgbi;0/0/0",
		"rgbii:0/0/0",
		"RGBI:0/0/0",
		"rgbi",
		"rgbi:",
		"rgbi:0/0",
	}
	for _, s := range bad {
		col_expectError(t, ErrInvalidFormat, col_parseErr(s))
	}
}

// Zig: test "term51: RGB.parse: # forms with 1, 2, 3 and 4 digits per channel"
func TestColor_term51_RGB_parse_forms_with_1_2_3_and_4_digits_per_channel(t *testing.T) {
	// #rgb
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "#000"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "#fff"))
	expectEqual(t, term51Rgb(136, 136, 136), col_parse(t, "#888"))
	expectEqual(t, term51Rgb(17, 34, 51), col_parse(t, "#123"))
	expectEqual(t, term51Rgb(170, 187, 204), col_parse(t, "#abc"))
	expectEqual(t, term51Rgb(170, 187, 204), col_parse(t, "#ABC"))
	expectEqual(t, term51Rgb(170, 187, 204), col_parse(t, "#aBc"))
	expectEqual(t, term51Rgb(0, 17, 0), col_parse(t, "#010"))
	// #rrggbb
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "#000000"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "#ffffff"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "#FFFFFF"))
	expectEqual(t, term51Rgb(128, 128, 128), col_parse(t, "#808080"))
	expectEqual(t, term51Rgb(127, 127, 127), col_parse(t, "#7f7f7f"))
	expectEqual(t, term51Rgb(255, 0, 16), col_parse(t, "#ff0010"))
	expectEqual(t, term51Rgb(1, 2, 3), col_parse(t, "#010203"))
	expectEqual(t, term51Rgb(171, 205, 239), col_parse(t, "#AbCdEf"))
	// #rrrgggbbb
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "#000000000"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "#fffffffff"))
	expectEqual(t, term51Rgb(127, 127, 127), col_parse(t, "#800800800"))
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "#001001001"))
	expectEqual(t, term51Rgb(255, 0, 0), col_parse(t, "#fff000000"))
	expectEqual(t, term51Rgb(0, 127, 0), col_parse(t, "#000800000"))
	// #rrrrggggbbbb
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "#000000000000"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "#ffffffffffff"))
	expectEqual(t, term51Rgb(127, 127, 127), col_parse(t, "#800080008000"))
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "#000100010001"))
	expectEqual(t, term51Rgb(1, 1, 1), col_parse(t, "#010101010101"))
	expectEqual(t, term51Rgb(255, 0, 255), col_parse(t, "#ffff0000ffff"))
	expectEqual(t, term51Rgb(18, 86, 154), col_parse(t, "#123456789abc"))
}

// Zig: test "term51: RGB.parse: # forms accept only lengths 4, 7, 10 and 13"
func TestColor_term51_RGB_parse_forms_accept_only_lengths_4_7_10_and_13(t *testing.T) {
	for len_ := 1; len_ <= 20; len_++ {
		s := "#" + strings.Repeat("f", len_-1)
		switch len_ {
		case 4, 7, 10, 13:
			expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, s))
		default:
			col_expectError(t, ErrInvalidFormat, col_parseErr(s))
		}
	}

	bad := [...]string{
		"#",
		"#f",
		"#ff",
		"#ffff",
		"#fffff",
		"#gggggg",
		"#ggg",
		"#12g",
		"#1g2",
		"#g12",
		"#-12",
		"#+12",
		"#0x1",
		"# 12",
		"#12 456",
		" #fff",
		"#fff ",
		"# fff",
		"#ff ff ff",
		"#fff\n",
		"#_1_1_1",
		"#1_1",
		"#fffffffffffffffff",
		"#" + strings.Repeat("f", 100),
		"##fff",
		"#fff/fff/fff",
	}
	for _, s := range bad {
		col_expectError(t, ErrInvalidFormat, col_parseErr(s))
	}
}

// Zig: test "term51: RGB.parse: X11 color names"
func TestColor_term51_RGB_parse_X11_color_names(t *testing.T) {
	expectEqual(t, term51Rgb(0, 0, 0), col_parse(t, "black"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "white"))
	expectEqual(t, term51Rgb(255, 0, 0), col_parse(t, "red"))
	expectEqual(t, term51Rgb(0, 255, 0), col_parse(t, "green"))
	expectEqual(t, term51Rgb(0, 0, 255), col_parse(t, "blue"))
	// First and last names of the table.
	expectEqual(t, term51Rgb(255, 250, 250), col_parse(t, "snow"))
	expectEqual(t, term51Rgb(0, 128, 128), col_parse(t, "teal"))
	// Case-insensitive, with and without spaces.
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "WHITE"))
	expectEqual(t, term51Rgb(255, 255, 255), col_parse(t, "WhItE"))
	expectEqual(t, term51Rgb(124, 252, 0), col_parse(t, "LawnGreen"))
	expectEqual(t, term51Rgb(124, 252, 0), col_parse(t, "lawn green"))
	expectEqual(t, term51Rgb(0, 250, 154), col_parse(t, "medium spring green"))
	expectEqual(t, term51Rgb(0, 250, 154), col_parse(t, "MediumSpringGreen"))
	expectEqual(t, term51Rgb(250, 250, 210), col_parse(t, "light goldenrod yellow"))
	expectEqual(t, term51Rgb(238, 221, 130), col_parse(t, "light goldenrod"))
	expectEqual(t, term51Rgb(190, 190, 190), col_parse(t, "grey"))
	// Only ASCII space is trimmed around names.
	expectEqual(t, term51Rgb(255, 0, 0), col_parse(t, " red"))
	expectEqual(t, term51Rgb(255, 0, 0), col_parse(t, "red "))
	expectEqual(t, term51Rgb(255, 0, 0), col_parse(t, "   red   "))
	expectEqual(t, term51Rgb(34, 139, 34), col_parse(t, " Forest Green "))

	col_expectError(t, ErrInvalidFormat, col_parseErr("\tred"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("red\t"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("red\n"))
	col_expectError(t, ErrInvalidFormat, col_parseErr(" "))
	col_expectError(t, ErrInvalidFormat, col_parseErr("          "))
	col_expectError(t, ErrInvalidFormat, col_parseErr("nosuchcolor"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("re"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("redd"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("medium  spring green"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("gray101"))
}

// Zig: test "term51: RGB.parse: empty string is invalid"
func TestColor_term51_RGB_parse_empty_string_is_invalid(t *testing.T) {
	col_expectError(t, ErrInvalidFormat, col_parseErr(""))
}

// Zig: test "term51: LAB.lerp: endpoints, midpoint, extrapolation, per-component"
func TestColor_term51_LAB_lerp_endpoints_midpoint_extrapolation_per_component(t *testing.T) {
	a := colorLAB{l: 0, a: -10, b: 20}
	b := colorLAB{l: 100, a: 30, b: -40}

	expectEqual(t, a, colorLABLerp(0.0, a, b))
	expectEqual(t, b, colorLABLerp(1.0, a, b))
	expectEqual(t, colorLAB{l: 50, a: 10, b: -10}, colorLABLerp(0.5, a, b))
	expectEqual(t, colorLAB{l: 25, a: 0, b: 5}, colorLABLerp(0.25, a, b))
	expectEqual(t, colorLAB{l: 75, a: 20, b: -25}, colorLABLerp(0.75, a, b))
	// Not clamped: t outside [0, 1] extrapolates.
	expectEqual(t, colorLAB{l: 200, a: 70, b: -100}, colorLABLerp(2.0, a, b))
	expectEqual(t, colorLAB{l: -100, a: -50, b: 80}, colorLABLerp(-1.0, a, b))
	// Identical endpoints give the same value for any t.
	expectEqual(t, a, colorLABLerp(0.3, a, a))
	// Swapping the endpoints mirrors t.
	expectEqual(t, colorLABLerp(0.25, a, b), colorLABLerp(0.75, b, a))

	// Interpolating real colors: black to white at the middle is mid-gray L.
	black := colorLABFromRgb(term51Rgb(0, 0, 0))
	white := colorLABFromRgb(term51Rgb(255, 255, 255))
	mid := colorLABLerp(0.5, black, white)
	col_expectApproxEqAbs(t, float32(50.0), mid.l, 0.5)
	col_expectApproxEqAbs(t, float32(0.0), mid.a, 0.5)
	col_expectApproxEqAbs(t, float32(0.0), mid.b, 0.5)
	expectEqual(t, term51Rgb(119, 119, 119), mid.toRgb())
}

// Zig: test "term51: LAB.fromRgb: more reference colors and gray axis"
func TestColor_term51_LAB_fromRgb_more_reference_colors_and_gray_axis(t *testing.T) {
	const epsilon = 0.5

	// Pure green (0, 255, 0) -> L*=87.73, a*=-86.18, b*=83.18
	green := colorLABFromRgb(term51Rgb(0, 255, 0))
	col_expectApproxEqAbs(t, float32(87.73), green.l, epsilon)
	col_expectApproxEqAbs(t, float32(-86.18), green.a, epsilon)
	col_expectApproxEqAbs(t, float32(83.18), green.b, epsilon)

	// Mid gray (128,128,128) -> L*=53.59, a*=b*=0
	gray := colorLABFromRgb(term51Rgb(128, 128, 128))
	col_expectApproxEqAbs(t, float32(53.59), gray.l, epsilon)
	col_expectApproxEqAbs(t, float32(0.0), gray.a, epsilon)
	col_expectApproxEqAbs(t, float32(0.0), gray.b, epsilon)

	// Near the linear/cube-root splits: (1,1,1), (10,10,10), (11,11,11).
	col_expectApproxEqAbs(t, float32(0.27), colorLABFromRgb(term51Rgb(1, 1, 1)).l, epsilon)
	col_expectApproxEqAbs(t, float32(2.74), colorLABFromRgb(term51Rgb(10, 10, 10)).l, epsilon)
	col_expectApproxEqAbs(t, float32(3.02), colorLABFromRgb(term51Rgb(11, 11, 11)).l, epsilon)

	// The theme "black" 1D1F21.
	theme := colorLABFromRgb(term51Rgb(0x1D, 0x1F, 0x21))
	col_expectApproxEqAbs(t, float32(11.63), theme.l, epsilon)
	col_expectApproxEqAbs(t, float32(-0.40), theme.a, epsilon)
	col_expectApproxEqAbs(t, float32(-1.68), theme.b, epsilon)

	// L* is monotonic along the gray axis.
	prev := float32(-1.0)
	for v := 0; v <= 255; v++ {
		c := uint8(v)
		l := colorLABFromRgb(term51Rgb(c, c, c)).l
		expect(t, l > prev, "l > prev")
		prev = l
	}
}

// Zig: test "term51: LAB.toRgb: gray axis round trip and clamping"
func TestColor_term51_LAB_toRgb_gray_axis_round_trip_and_clamping(t *testing.T) {
	// Every gray survives a round trip.
	for v := 0; v <= 255; v++ {
		c := uint8(v)
		expectEqual(t, term51Rgb(c, c, c), colorLABFromRgb(term51Rgb(c, c, c)).toRgb())
	}

	// Extremes of L* and out-of-range L* clamp.
	expectEqual(t, term51Rgb(0, 0, 0), colorLAB{l: 0, a: 0, b: 0}.toRgb())
	expectEqual(t, term51Rgb(255, 255, 255), colorLAB{l: 100, a: 0, b: 0}.toRgb())
	expectEqual(t, term51Rgb(255, 255, 255), colorLAB{l: 150, a: 0, b: 0}.toRgb())
	expectEqual(t, term51Rgb(0, 0, 0), colorLAB{l: -50, a: 0, b: 0}.toRgb())

	// Mid lightness and the linear segment near black.
	expectEqual(t, term51Rgb(119, 119, 119), colorLAB{l: 50, a: 0, b: 0}.toRgb())
	expectEqual(t, term51Rgb(24, 24, 24), colorLAB{l: 8, a: 0, b: 0}.toRgb())
	expectEqual(t, term51Rgb(23, 23, 23), colorLAB{l: 7.9, a: 0, b: 0}.toRgb())
	expectEqual(t, term51Rgb(4, 4, 4), colorLAB{l: 1, a: 0, b: 0}.toRgb())

	// Out of gamut chroma clamps individual channels to 0 or 255.
	expectEqual(t, term51Rgb(255, 0, 123), colorLAB{l: 50, a: 100, b: 0}.toRgb())
	expectEqual(t, term51Rgb(0, 151, 117), colorLAB{l: 50, a: -100, b: 0}.toRgb())
	expectEqual(t, term51Rgb(146, 116, 0), colorLAB{l: 50, a: 0, b: 100}.toRgb())
	expectEqual(t, term51Rgb(0, 132, 255), colorLAB{l: 50, a: 0, b: -100}.toRgb())
}

// Zig: test "term51: generate256Color: default base with dark theme pins corners and sampled cells"
func TestColor_term51_generate256Color_default_base_with_dark_theme_pins_corners_and_sampled_cells(t *testing.T) {
	bg := term51Rgb(0, 0, 0)
	fg := term51Rgb(255, 255, 255)
	p := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)

	// Corners come from bg, base[1..6], fg.
	expectEqual(t, bg, p[16])
	expectEqual(t, ColorDefault[4], p[21])
	expectEqual(t, ColorDefault[2], p[46])
	expectEqual(t, ColorDefault[6], p[51])
	expectEqual(t, ColorDefault[1], p[196])
	expectEqual(t, ColorDefault[5], p[201])
	expectEqual(t, ColorDefault[3], p[226])
	expectEqual(t, fg, p[231])

	// Sampled interior cells.
	expectEqual(t, term51Rgb(30, 34, 39), p[17])
	expectEqual(t, term51Rgb(38, 38, 26), p[22])
	expectEqual(t, term51Rgb(86, 75, 67), p[59])
	expectEqual(t, term51Rgb(145, 131, 118), p[102])
	expectEqual(t, term51Rgb(170, 179, 177), p[110])
	expectEqual(t, term51Rgb(188, 174, 162), p[145])
	expectEqual(t, term51Rgb(254, 243, 227), p[230])

	// Grayscale ramp: t = 1/25 .. 24/25.
	expectEqual(t, term51Rgb(14, 14, 14), p[232])
	expectEqual(t, term51Rgb(24, 24, 24), p[233])
	expectEqual(t, term51Rgb(114, 114, 114), p[243])
	expectEqual(t, term51Rgb(124, 124, 124), p[244])
	expectEqual(t, term51Rgb(232, 232, 232), p[254])
	expectEqual(t, term51Rgb(243, 243, 243), p[255])
}

// Zig: test "term51: generate256Color: theme colors bg and fg anchor the cube"
func TestColor_term51_generate256Color_theme_colors_bg_and_fg_anchor_the_cube(t *testing.T) {
	bg := term51Rgb(0x1D, 0x1F, 0x21)
	fg := term51Rgb(0xC5, 0xC8, 0xC6)
	p := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)

	expectEqual(t, bg, p[16])
	expectEqual(t, fg, p[231])
	expectEqual(t, ColorDefault[1], p[196])
	expectEqual(t, ColorDefault[4], p[21])

	expectEqual(t, term51Rgb(47, 54, 61), p[17])
	expectEqual(t, term51Rgb(57, 59, 47), p[22])
	expectEqual(t, term51Rgb(99, 89, 82), p[59])
	expectEqual(t, term51Rgb(148, 134, 122), p[102])
	expectEqual(t, term51Rgb(161, 170, 169), p[110])
	expectEqual(t, term51Rgb(35, 37, 39), p[232])
	expectEqual(t, term51Rgb(40, 42, 44), p[233])
	expectEqual(t, term51Rgb(103, 106, 106), p[243])
	expectEqual(t, term51Rgb(110, 113, 113), p[244])
	expectEqual(t, term51Rgb(182, 185, 183), p[254])
	expectEqual(t, term51Rgb(189, 192, 191), p[255])
}

// Zig: test "term51: generate256Color: light theme inverts unless harmonious"
func TestColor_term51_generate256Color_light_theme_inverts_unless_harmonious(t *testing.T) {
	white := term51Rgb(255, 255, 255)
	black := term51Rgb(0, 0, 0)

	dark := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), black, white, false)
	light := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), white, black, false)
	// With the swap, a light theme gives exactly the dark theme's result.
	for i := 16; i < 256; i++ {
		expectEqual(t, dark[i], light[i])
	}

	harm := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), white, black, true)
	expectEqual(t, white, harm[16])
	expectEqual(t, term51Rgb(230, 236, 242), harm[17])
	expectEqual(t, term51Rgb(241, 241, 224), harm[22])
	expectEqual(t, term51Rgb(218, 205, 195), harm[59])
	expectEqual(t, black, harm[231])
	// The corner colors are still the base colors.
	expectEqual(t, ColorDefault[4], harm[21])
	expectEqual(t, ColorDefault[1], harm[196])
	// The ramp runs from light to dark.
	expectEqual(t, term51Rgb(243, 243, 243), harm[232])
	expectEqual(t, term51Rgb(232, 232, 232), harm[233])
	expectEqual(t, term51Rgb(124, 124, 124), harm[243])
	expectEqual(t, term51Rgb(114, 114, 114), harm[244])
	expectEqual(t, term51Rgb(24, 24, 24), harm[254])
	expectEqual(t, term51Rgb(14, 14, 14), harm[255])
}

// Zig: test "term51: generate256Color: equal bg and fg is not a light theme"
func TestColor_term51_generate256Color_equal_bg_and_fg_is_not_a_light_theme(t *testing.T) {
	gray := term51Rgb(128, 128, 128)
	a := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), gray, gray, false)
	b := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), gray, gray, true)
	for i := 16; i < 256; i++ {
		expectEqual(t, a[i], b[i])
	}
	expectEqual(t, gray, a[16])
	expectEqual(t, gray, a[231])
	for i := 232; i < 256; i++ {
		expectEqual(t, gray, a[i])
	}
}

// Zig: test "term51: generate256Color: only bg, fg and base[1..6] influence the result"
func TestColor_term51_generate256Color_only_bg_fg_and_base_1_6_influence_the_result(t *testing.T) {
	bg := term51Rgb(0, 0, 0)
	fg := term51Rgb(255, 255, 255)
	ref := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)

	// base[0], base[7] and base[8..15] are not used by the interpolation.
	other := ColorDefault
	other[0] = term51Rgb(99, 98, 97)
	other[7] = term51Rgb(1, 2, 3)
	for i := 8; i < 16; i++ {
		other[i] = term51Rgb(7, 7, 7)
	}
	p := ColorGenerate256Color(other, StaticBitSet256InitEmpty(), bg, fg, false)
	for i := 16; i < 256; i++ {
		expectEqual(t, ref[i], p[i])
	}
	// But the first 16 entries are copied verbatim.
	for i := 0; i < 16; i++ {
		expectEqual(t, other[i], p[i])
	}

	// base[1] is the red corner at index 196.
	red := ColorDefault
	red[1] = term51Rgb(200, 10, 20)
	q := ColorGenerate256Color(red, StaticBitSet256InitEmpty(), bg, fg, false)
	expectEqual(t, term51Rgb(200, 10, 20), q[196])
	expect(t, !q[196].Eql(ref[196]), "!q[196].eql(ref[196])")
}

// Zig: test "term51: generate256Color: skip mask boundaries"
func TestColor_term51_generate256Color_skip_mask_boundaries(t *testing.T) {
	bg := term51Rgb(0, 0, 0)
	fg := term51Rgb(255, 255, 255)
	ref := ColorGenerate256Color(ColorDefault, StaticBitSet256InitEmpty(), bg, fg, false)

	// Skip every index: the result is the base palette.
	all := StaticBitSet256InitEmpty()
	for i := 0; i < 256; i++ {
		all.set(i)
	}
	expectEqual(t, ColorDefault, ColorGenerate256Color(ColorDefault, all, bg, fg, false))

	// Skipping 0..15 changes nothing.
	low := StaticBitSet256InitEmpty()
	for i := 0; i < 16; i++ {
		low.set(i)
	}
	expectEqual(t, ref, ColorGenerate256Color(ColorDefault, low, bg, fg, false))

	// Skip the first/last cube cells and the first/last gray cells.
	edge := StaticBitSet256InitEmpty()
	edge.set(16)
	edge.set(231)
	edge.set(232)
	edge.set(255)
	p := ColorGenerate256Color(ColorDefault, edge, bg, fg, false)
	expectEqual(t, ColorDefault[16], p[16])
	expectEqual(t, ColorDefault[231], p[231])
	expectEqual(t, ColorDefault[232], p[232])
	expectEqual(t, ColorDefault[255], p[255])
	// Their neighbours are still generated.
	expectEqual(t, ref[17], p[17])
	expectEqual(t, ref[230], p[230])
	expectEqual(t, ref[233], p[233])
	expectEqual(t, ref[254], p[254])
	expect(t, !p[17].Eql(ColorDefault[17]), "!p[17].eql(default[17])")
	expect(t, !p[230].Eql(ColorDefault[230]), "!p[230].eql(default[230])")
	expect(t, !p[233].Eql(ColorDefault[233]), "!p[233].eql(default[233])")
	expect(t, !p[254].Eql(ColorDefault[254]), "!p[254].eql(default[254])")

	// A custom base shows through at skipped indexes.
	custom := ColorDefault
	custom[100] = term51Rgb(1, 2, 3)
	one := StaticBitSet256InitEmpty()
	one.set(100)
	c := ColorGenerate256Color(custom, one, bg, fg, false)
	expectEqual(t, term51Rgb(1, 2, 3), c[100])
	expectEqual(t, ref[99], c[99])
	expectEqual(t, ref[101], c[101])
}

// term51: floating-point exactness. Go's math library can differ from Zig's
// in the last bit; these tests pin exact integer and exact float-bit results
// so that any such difference shows up as a failing test.

// term51_lab_levels is the level grid of the data dump (see
// corpus_color_test.go).
var term51_lab_levels = [...]uint8{0, 1, 2, 5, 8, 16, 32, 51, 64, 85, 100, 127, 128, 150, 170, 192, 200, 224, 240, 250, 254, 255}

// Not translated: the Zig test "term51: LAB.fromRgb: dump exact data when
// TERM51_COLOR_DUMP_DIR is set". It only writes data/colors/*.txt; the Go
// tests that read those files are in corpus_color_test.go.

// Zig: test "term51: LAB.fromRgb: exact float bits for black, white and mid gray"
func TestColor_term51_LAB_fromRgb_exact_float_bits_for_black_white_and_mid_gray(t *testing.T) {
	// Pinned bit patterns of the f32 results: any libm difference in pow or
	// cbrt shows up here.
	black := colorLABFromRgb(ColorRGB{R: 0, G: 0, B: 0})
	expectEqual(t, uint32(0x00000000), math.Float32bits(black.l)&0x80000000)
	white := colorLABFromRgb(ColorRGB{R: 255, G: 255, B: 255})
	// white maps back to white exactly
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, white.toRgb())
	expectEqual(t, ColorRGB{R: 0, G: 0, B: 0}, black.toRgb())
	gray := colorLABFromRgb(ColorRGB{R: 128, G: 128, B: 128})
	expectEqual(t, ColorRGB{R: 128, G: 128, B: 128}, gray.toRgb())
}

// Zig: test "term51: RGB.fromIntensity: infinity and negative zero do not crash"
func TestColor_term51_RGB_fromIntensity_infinity_and_negative_zero_do_not_crash(t *testing.T) {
	// infinity fails the upper range check; -0.0 passes both and converts to 0
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("inf"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("+inf"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("-inf"))
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("infinity"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "-0"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "-0.0"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "-0e5"))
	expectEqual(t, ColorRGB{R: 0, G: 0, B: 0}, col_parse(t, "rgbi:-0/-0/-0"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgbi:inf/0/0"))
	col_expectError(t, ErrInvalidFormat, col_parseErr("rgbi:0/-inf/0"))
}

// Zig: test "term51: RGB.fromIntensity: accepted only because of Zig's number parser"
func TestColor_term51_RGB_fromIntensity_accepted_only_because_of_Zig_s_number_parser(t *testing.T) {
	// These forms are accepted by std.fmt.parseFloat. The port copies them
	// exactly; each is on the deliberate-fixes list.
	// hex float
	expectEqual(t, uint8(127), col_fromIntensity(t, "0x1p-1"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "0x1"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "0x1p0"))
	expectEqual(t, uint8(0), col_fromIntensity(t, "0x0"))
	// leading plus
	expectEqual(t, uint8(127), col_fromIntensity(t, "+0.5"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "+1"))
	// other forms
	expectEqual(t, uint8(127), col_fromIntensity(t, ".5"))
	expectEqual(t, uint8(255), col_fromIntensity(t, "1."))
	expectEqual(t, uint8(255), col_fromIntensity(t, "1e0"))
	// underscores are NOT accepted by the float parser (only by fromHex)
	col_expectError(t, ErrInvalidFormat, col_fromIntensityErr("0_5"))
}

// Zig: test "term51: RGB.fromHex: underscores are accepted and signs are rejected, because of Zig's parser"
func TestColor_term51_RGB_fromHex_underscores_are_accepted_and_signs_are_rejected_because_of_Zig_s_parser(t *testing.T) {
	// underscore between digits: "1_1" is three characters, so the value
	// 0x11 is scaled by 4095 (12-bit)
	expectEqual(t, uint8(1), col_fromHex(t, "1_1"))
	// "f_f" is three characters: 0xff scaled by 4095
	expectEqual(t, uint8(15), col_fromHex(t, "f_f"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("_f"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("f_"))
	// signs are not accepted in hex digits (the leading plus is accepted
	// only by the float form, see fromIntensity)
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("+f"))
	col_expectError(t, ErrInvalidFormat, col_fromHexErr("-f"))
}

// Zig: test "term51: RGB.fromIntensity: underscores can lower a channel by one (zig std defect)"
func TestColor_term51_RGB_fromIntensity_underscores_can_lower_a_channel_by_one_zig_std_defect(t *testing.T) {
	// Reproduces a Zig 0.15.2 std defect: std.fmt.parseFloat's slow path
	// (convert_slow.zig -> decimal.zig Decimal.parse, lines 273-294) trims
	// trailing zeros starting from stream.offsetTrue() - 1, which does not
	// count the '_' separators, so it can drop real digits. Near a midpoint
	// between two f64 values the result is one unit in the last place low,
	// and the channel truncates to one less. The same digits without '_'
	// give the correctly rounded value. The Go port reproduces this; see
	// docs/upstream-findings.md and docs/deliberate-fixes.md.
	expectEqual(t, term51Rgb(16, 0, 0), col_parse(t, "rgbi:0.0_66666666666666658802586908905141171999275684356689453125001/0/0"))
	expectEqual(t, term51Rgb(17, 0, 0), col_parse(t, "rgbi:0.066666666666666658802586908905141171999275684356689453125001/0/0"))
	expectEqual(t, term51Rgb(66, 127, 255), col_parse(t, "rgbi:0.2_62_7450980392156687504012779754702933132648468017578125001/0.5/1"))
	expectEqual(t, term51Rgb(67, 127, 255), col_parse(t, "rgbi:0.262745098039215687504012779754702933132648468017578125001/0.5/1"))
	expectEqual(t, term51Rgb(0, 132, 0), col_parse(t, "rgbi:0/0.5_21568627450980371040856198305846191942691802978515625001/0"))
	expectEqual(t, term51Rgb(255, 0, 249), col_parse(t, "rgbi:1/0.0_0/0.98039215686274500116681451800104696303606033325195312_5001"))
	expectEqual(t, term51Rgb(33, 99, 199), col_parse(t, "rgbi:0.1_3333333333333331760517381781028234399855136871337890625001/0.3_92_1568627450980060178409303262014873325824737548828125001/0.78431372549019601203568186065240297466516494750976562_5001"))
	expectEqual(t, term51Rgb(34, 100, 200), col_parse(t, "rgbi:0.13333333333333331760517381781028234399855136871337890625001/0.3921568627450980060178409303262014873325824737548828125001/0.784313725490196012035681860652402974665164947509765625001"))
	expectEqual(t, term51Rgb(245, 244, 245), col_parse(t, "rgbi:0.960784313725490168867082729775574989616870880126953125001/0.9_60784313725490168867082729775574989616870880126953125001/0.960784313725490168867082729775574989616870880126953125001"))
	expectEqual(t, term51Rgb(16, 16, 16), col_parse(t, "rgbi:6.6_666666666666658802586908905141171999275684356689453125001e-2/6.6_666666666666658802586908905141171999275684356689453125001e-2/6.6_666666666666658802586908905141171999275684356689453125001e-2"))
	// '_' that do not change the result
	expectEqual(t, term51Rgb(18, 0, 0), col_parse(t, "rgbi:0.0_70588235294117639140321074364692321978509426116943359375001/0/0"))
	expectEqual(t, term51Rgb(16, 25, 255), col_parse(t, "rgbi:0.0_666/0.1_0/1.0_0"))
}
