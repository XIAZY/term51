// Port of src/terminal/color.zig (Ghostty v1.3.1).
//
// Declarations appear in the same order as in color.zig. The C ABI
// declarations (`C`, `cval`) are not ported; see docs/mapping.md.
//
// Floating point. Every f32 and f64 computation follows Zig's operation
// order with Go's float32/float64. Go may fuse a multiply and a following
// add into one instruction (it does on arm64); Zig does not. Every product
// that is an operand of an addition or subtraction is therefore wrapped in
// an explicit float32(...) or float64(...) conversion, which the Go
// specification says rounds and prevents the fusion. Zig's std.math.pow,
// std.math.cbrt, std.math.ldexp, std.fmt.parseUnsigned and the compiler_rt
// exp/log functions that @exp and @log call are ported at the end of this
// file: Go's math package computes in float64 and differs in the last bit.
// std.fmt.parseFloat is ported in std_fmt_parse_float.go: strconv's grammar
// is not Zig's, and Zig's result is not always correctly rounded.

package terminal

import (
	"errors"
	"math"
	"math/bits"
)

// notImplemented is the panic value of every wireframe function body.
const notImplemented = "wireframe: not implemented"

// ErrNoDefaultValue is Zig's error.NoDefaultValue (color.Name.default).
var ErrNoDefaultValue = errors.New("NoDefaultValue")

// ErrInvalidFormat is Zig's error.InvalidFormat (color.RGB.parse and its
// helpers).
var ErrInvalidFormat = errors.New("InvalidFormat")

// ErrNoSpaceLeft is Zig's error.NoSpaceLeft, returned by fixed-buffer
// writers and std.fmt.bufPrint when the buffer is too small. It is declared
// here with the other shared error values of the wireframe.
var ErrNoSpaceLeft = errors.New("NoSpaceLeft")

// ColorDefault is color.default: the default palette.
//
// Zig computes it at compile time: the 16 named colors from Name.default,
// the 6x6x6 cube, then the grayscale ramp. Go computes it once at start-up
// with colorDefaultInit.
var ColorDefault ColorPalette = colorDefaultInit()

// colorDefaultInit is the comptime block that initialises color.default.
func colorDefaultInit() ColorPalette {
	var result ColorPalette

	// Named values
	var i uint8 = 0
	for i < 16 {
		v, err := ColorName(i).Default()
		if err != nil {
			panic("unreachable")
		}
		result[i] = v
		i += 1
	}

	// Cube
	assert(i == 16)
	var r uint8 = 0
	for r < 6 {
		var g uint8 = 0
		for g < 6 {
			var b uint8 = 0
			for b < 6 {
				result[i] = ColorRGB{
					R: colorDefaultCubeLevel(r),
					G: colorDefaultCubeLevel(g),
					B: colorDefaultCubeLevel(b),
				}

				i += 1
				b += 1
			}
			g += 1
		}
		r += 1
	}

	// Gray ramp
	assert(i == 232)
	// Zig: assert(@TypeOf(i) == u8); i is a uint8, so `i +%= 1` wraps to 0
	// after 255 and ends the loop.
	for i > 0 {
		value := ((i - 232) * 10) + 8
		result[i] = ColorRGB{R: value, G: value, B: value}
		i += 1
	}

	return result
}

// colorDefaultCubeLevel is `if (c == 0) 0 else (c * 40 + 55)` of the cube
// loop in color.default (Go has no conditional expression).
func colorDefaultCubeLevel(c uint8) uint8 {
	if c == 0 {
		return 0
	}
	return c*40 + 55
}

// ColorPalette is color.Palette: palette is the 256 color palette.
type ColorPalette [256]ColorRGB

// ColorPaletteMask is color.PaletteMask: mask that can be used to set which
// palette indexes were set.
//
// Zig: std.StaticBitSet(256).
type ColorPaletteMask = StaticBitSet256

// StaticBitSet256 is std.StaticBitSet(256). It belongs in zig.go beside
// StaticBitSet24; it is declared here so that this port adds no code to a
// file another port is editing. Methods keep Zig's names.
//
// Zig: std.StaticBitSet(256) is an ArrayBitSet(usize, 256): four 64-bit
// masks, bit i in masks[i/64] at position i%64.
type StaticBitSet256 struct {
	masks [4]uint64
}

// StaticBitSet256BitLength is std.StaticBitSet(256).bit_length.
const StaticBitSet256BitLength = 256

// StaticBitSet256InitEmpty is std.StaticBitSet(256).initEmpty.
func StaticBitSet256InitEmpty() StaticBitSet256 {
	return StaticBitSet256{}
}

// StaticBitSet256InitFull is std.StaticBitSet(256).initFull.
func StaticBitSet256InitFull() StaticBitSet256 {
	return StaticBitSet256{masks: [4]uint64{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64}}
}

// isSet reports whether the bit at index is set. Zig asserts
// index < bit_length.
func (self StaticBitSet256) isSet(index int) bool {
	assert(index >= 0 && index < StaticBitSet256BitLength)
	return self.masks[index/64]&(uint64(1)<<uint(index%64)) != 0
}

// set sets the bit at index. Zig asserts index < bit_length.
func (self *StaticBitSet256) set(index int) {
	assert(index >= 0 && index < StaticBitSet256BitLength)
	self.masks[index/64] |= uint64(1) << uint(index%64)
}

// unset clears the bit at index. Zig asserts index < bit_length.
func (self *StaticBitSet256) unset(index int) {
	assert(index >= 0 && index < StaticBitSet256BitLength)
	self.masks[index/64] &^= uint64(1) << uint(index%64)
}

// count returns the number of set bits.
func (self StaticBitSet256) count() int {
	total := 0
	for _, m := range self.masks {
		total += bits.OnesCount64(m)
	}
	return total
}

// iterator returns an iterator over the indexes of the set bits, in
// ascending order. Zig: iterator(.{}).
func (self *StaticBitSet256) iterator() StaticBitSet256Iterator {
	return StaticBitSet256Iterator{set: self, next: 0}
}

// StaticBitSet256Iterator is the iterator returned by
// StaticBitSet256.iterator.
//
// Zig's BitSetIterator copies the current mask word and reads the later
// words from the set as it reaches them; this one reads the set at each
// step. The two differ only if the set changes during the iteration, which
// no ported code does.
type StaticBitSet256Iterator struct {
	set  *StaticBitSet256
	next int
}

// Next returns the next set index, or none at the end. Zig: next() ?usize.
func (self *StaticBitSet256Iterator) Next() Optional[int] {
	for self.next < StaticBitSet256BitLength {
		i := self.next
		self.next += 1
		if self.set.isSet(i) {
			return Some(i)
		}
	}
	return None[int]()
}

// ColorGenerate256Color is color.generate256Color: generate the 256-color
// palette from the user's base16 theme colors, terminal background, and
// terminal foreground.
//
// Motivation: The default 256-color palette uses fixed, fully-saturated
// colors that clash with custom base16 themes, have poor readability in
// dark shades (the first non-black shade jumps to 37% intensity instead
// of the expected 20%), and exhibit inconsistent perceived brightness
// across hues of the same shade (e.g., blue appears darker than green).
// By generating the extended palette from the user's chosen colors,
// programs can use the richer 256-color range without requiring their
// own theme configuration, and light/dark switching works automatically.
//
// The 216-color cube (indices 16–231) is built via trilinear
// interpolation in CIELAB space over the 8 base colors. The base16
// palette maps to the 8 corners of a 6×6×6 RGB cube as follows:
//
//	R=0 edge: bg      → base[1] (red)
//	R=5 edge: base[6] → fg
//	G=0 edge: bg/base[6] (via R) → base[2]/base[4] (green/blue via R)
//	G=5 edge: base[1]/fg (via R) → base[3]/base[5] (yellow/magenta via R)
//
// For each R slice, four corner colors (c0–c3) are interpolated along
// the R axis, then for each G row two edge colors (c4–c5) are
// interpolated along G, and finally each B cell is interpolated along B
// to produce the final color. CIELAB interpolation ensures perceptually
// uniform brightness transitions across different hues.
//
// The 24-step grayscale ramp (indices 232–255) is a simple linear
// interpolation in CIELAB from the background to the foreground,
// excluding pure black and white (available in the cube at (0,0,0)
// and (5,5,5)). The interpolation parameter runs from 1/25 to 24/25.
//
// Fill `skip` with user-defined color indexes to avoid replacing them.
//
// Reference: https://gist.github.com/jake-stewart/0a8ea46159a7da2c808e5be2177e1783
func ColorGenerate256Color(
	base ColorPalette,
	skip ColorPaletteMask,
	bg ColorRGB,
	fg ColorRGB,
	harmonious bool,
) ColorPalette {
	// Convert the background, foreground, and 8 base theme colors into
	// CIELAB space so that all interpolation is perceptually uniform.
	base8_lab := func() [8]colorLAB {
		base8 := [8]colorLAB{
			colorLABFromRgb(bg),
			colorLABFromRgb(base[1]),
			colorLABFromRgb(base[2]),
			colorLABFromRgb(base[3]),
			colorLABFromRgb(base[4]),
			colorLABFromRgb(base[5]),
			colorLABFromRgb(base[6]),
			colorLABFromRgb(fg),
		}

		// For light themes (where the foreground is darker than the
		// background), the cube's dark-to-light orientation is inverted
		// relative to the base color mapping. When `harmonious` is false,
		// swap bg and fg so the cube still runs from black (16) to
		// white (231).
		is_light_theme := base8[7].l < base8[0].l
		invert := is_light_theme && !harmonious
		if invert {
			base8[0], base8[7] = base8[7], base8[0]
		}

		return base8
	}()

	// Start from the base palette so indices 0–15 are preserved as-is.
	result := base

	// Build the 216-color cube (indices 16–231) via trilinear interpolation
	// in CIELAB. The three nested loops correspond to the R, G, and B axes
	// of a 6×6×6 cube. For each R slice, four corner colors (c0–c3) are
	// interpolated along R from the 8 base colors, mapping the cube corners
	// to theme-aware anchors (see doc comment for the mapping). Then for
	// each G row, two edge colors (c4–c5) blend along G, and finally each
	// B cell interpolates along B to produce the final color.
	var idx uint = 16
	for ri := uint(0); ri < 6; ri++ {
		// R-axis corners: blend base colors along the red dimension.
		tr := float32(ri) / 5.0
		c0 := colorLABLerp(tr, base8_lab[0], base8_lab[1])
		c1 := colorLABLerp(tr, base8_lab[2], base8_lab[3])
		c2 := colorLABLerp(tr, base8_lab[4], base8_lab[5])
		c3 := colorLABLerp(tr, base8_lab[6], base8_lab[7])
		for gi := uint(0); gi < 6; gi++ {
			// G-axis edges: blend the R-interpolated corners along green.
			tg := float32(gi) / 5.0
			c4 := colorLABLerp(tg, c0, c1)
			c5 := colorLABLerp(tg, c2, c3)
			for bi := uint(0); bi < 6; bi++ {
				// B-axis: final interpolation along blue, then convert back to RGB.
				if !skip.isSet(int(idx)) {
					c6 := colorLABLerp(
						float32(bi)/5.0,
						c4,
						c5,
					)
					result[idx] = c6.toRgb()
				}

				idx += 1
			}
		}
	}

	// Build the 24-step grayscale ramp (indices 232–255) by linearly
	// interpolating in CIELAB from background to foreground. The parameter
	// runs from 1/25 to 24/25, excluding the endpoints which are already
	// available in the cube at (0,0,0) and (5,5,5).
	for i := uint(0); i < 24; i++ {
		t := float32(i+1) / 25.0
		if !skip.isSet(int(idx)) {
			c := colorLABLerp(t, base8_lab[0], base8_lab[7])
			result[idx] = c.toRgb()
		}
		idx += 1
	}

	return result
}

// ColorDynamicPalette is color.DynamicPalette: a palette that can have its
// colors changed and reset. Purposely built for terminal color operations.
type ColorDynamicPalette struct {
	// The current palette including any user modifications.
	Current ColorPalette

	// The original/default palette values.
	Original ColorPalette

	// A bitset where each bit represents whether the corresponding
	// palette index has been modified from its default value.
	Mask ColorPaletteMask
}

// ColorDynamicPaletteDefault is color.DynamicPalette.default:
// init(color.default), computed once at start-up.
var ColorDynamicPaletteDefault ColorDynamicPalette = ColorDynamicPaletteInit(ColorDefault)

// ColorDynamicPaletteInit is color.DynamicPalette.init: initialize a dynamic
// palette with a default palette.
func ColorDynamicPaletteInit(def ColorPalette) ColorDynamicPalette {
	return ColorDynamicPalette{
		Current:  def,
		Original: def,
		Mask:     StaticBitSet256InitEmpty(),
	}
}

// Set is color.DynamicPalette.set: set a custom color at the given palette
// index.
func (self *ColorDynamicPalette) Set(idx uint8, color ColorRGB) {
	self.Current[idx] = color
	self.Mask.set(int(idx))
}

// Reset is color.DynamicPalette.reset: reset the color at the given palette
// index to its original value.
func (self *ColorDynamicPalette) Reset(idx uint8) {
	self.Current[idx] = self.Original[idx]
	self.Mask.unset(int(idx))
}

// ResetAll is color.DynamicPalette.resetAll: reset all colors to their
// original values.
func (self *ColorDynamicPalette) ResetAll() {
	*self = ColorDynamicPaletteInit(self.Original)
}

// ChangeDefault is color.DynamicPalette.changeDefault: change the default
// palette, but preserve the changed values.
func (self *ColorDynamicPalette) ChangeDefault(def ColorPalette) {
	self.Original = def

	// Fast path, the palette is usually not changed.
	if self.Mask.count() == 0 {
		self.Current = self.Original
		return
	}

	// There are usually less set than unset, so iterate over the changed
	// values and override them.
	current := def
	it := self.Mask.iterator()
	for idx := it.Next(); idx.Valid; idx = it.Next() {
		current[idx.Value] = self.Current[idx.Value]
	}
	self.Current = current
}

// ColorDynamicRGB is color.DynamicRGB: RGB value that can be changed and
// reset. This can also be totally unset in every way, in which case the
// caller can determine their own ultimate default.
type ColorDynamicRGB struct {
	Override Optional[ColorRGB]
	Default  Optional[ColorRGB]
}

// ColorDynamicRGBUnset is color.DynamicRGB.unset: both fields null. The zero
// value is already that state.
var ColorDynamicRGBUnset = ColorDynamicRGB{}

// ColorDynamicRGBInit is color.DynamicRGB.init.
func ColorDynamicRGBInit(def ColorRGB) ColorDynamicRGB {
	return ColorDynamicRGB{
		Override: None[ColorRGB](),
		Default:  Some(def),
	}
}

// Get is color.DynamicRGB.get.
func (self *ColorDynamicRGB) Get() Optional[ColorRGB] {
	// Zig: self.override orelse self.default
	if self.Override.Valid {
		return self.Override
	}
	return self.Default
}

// Set is color.DynamicRGB.set.
func (self *ColorDynamicRGB) Set(color ColorRGB) {
	self.Override = Some(color)
}

// Reset is color.DynamicRGB.reset.
func (self *ColorDynamicRGB) Reset() {
	self.Override = self.Default
}

// ColorName is color.Name: color names in the standard 8 or 16 color
// palette.
//
// Zig: a non-exhaustive enum(u8); values 16 to 255 are valid and unnamed.
type ColorName uint8

const (
	ColorNameBlack   ColorName = 0
	ColorNameRed     ColorName = 1
	ColorNameGreen   ColorName = 2
	ColorNameYellow  ColorName = 3
	ColorNameBlue    ColorName = 4
	ColorNameMagenta ColorName = 5
	ColorNameCyan    ColorName = 6
	ColorNameWhite   ColorName = 7

	ColorNameBrightBlack   ColorName = 8
	ColorNameBrightRed     ColorName = 9
	ColorNameBrightGreen   ColorName = 10
	ColorNameBrightYellow  ColorName = 11
	ColorNameBrightBlue    ColorName = 12
	ColorNameBrightMagenta ColorName = 13
	ColorNameBrightCyan    ColorName = 14
	ColorNameBrightWhite   ColorName = 15

	// Remainders are valid unnamed values in the 256 color palette.
)

// colorNameNames are the tag names of the named ColorName values, 0 to 15.
var colorNameNames = [16]string{
	"black",
	"red",
	"green",
	"yellow",
	"blue",
	"magenta",
	"cyan",
	"white",
	"bright_black",
	"bright_red",
	"bright_green",
	"bright_yellow",
	"bright_blue",
	"bright_magenta",
	"bright_cyan",
	"bright_white",
}

// String returns the Zig tag name (@tagName).
//
// Unnamed values (16 to 255) return ""; dumps print them as #N.
func (self ColorName) String() string {
	if int(self) < len(colorNameNames) {
		return colorNameNames[self]
	}
	return ""
}

// Default is color.Name.default: default colors for tagged values. Returns
// ErrNoDefaultValue for unnamed values.
func (self ColorName) Default() (ColorRGB, error) {
	switch self {
	case ColorNameBlack:
		return ColorRGB{R: 0x1D, G: 0x1F, B: 0x21}, nil
	case ColorNameRed:
		return ColorRGB{R: 0xCC, G: 0x66, B: 0x66}, nil
	case ColorNameGreen:
		return ColorRGB{R: 0xB5, G: 0xBD, B: 0x68}, nil
	case ColorNameYellow:
		return ColorRGB{R: 0xF0, G: 0xC6, B: 0x74}, nil
	case ColorNameBlue:
		return ColorRGB{R: 0x81, G: 0xA2, B: 0xBE}, nil
	case ColorNameMagenta:
		return ColorRGB{R: 0xB2, G: 0x94, B: 0xBB}, nil
	case ColorNameCyan:
		return ColorRGB{R: 0x8A, G: 0xBE, B: 0xB7}, nil
	case ColorNameWhite:
		return ColorRGB{R: 0xC5, G: 0xC8, B: 0xC6}, nil

	case ColorNameBrightBlack:
		return ColorRGB{R: 0x66, G: 0x66, B: 0x66}, nil
	case ColorNameBrightRed:
		return ColorRGB{R: 0xD5, G: 0x4E, B: 0x53}, nil
	case ColorNameBrightGreen:
		return ColorRGB{R: 0xB9, G: 0xCA, B: 0x4A}, nil
	case ColorNameBrightYellow:
		return ColorRGB{R: 0xE7, G: 0xC5, B: 0x47}, nil
	case ColorNameBrightBlue:
		return ColorRGB{R: 0x7A, G: 0xA6, B: 0xDA}, nil
	case ColorNameBrightMagenta:
		return ColorRGB{R: 0xC3, G: 0x97, B: 0xD8}, nil
	case ColorNameBrightCyan:
		return ColorRGB{R: 0x70, G: 0xC0, B: 0xB1}, nil
	case ColorNameBrightWhite:
		return ColorRGB{R: 0xEA, G: 0xEA, B: 0xEA}, nil

	default:
		return ColorRGB{}, ErrNoDefaultValue
	}
}

// ColorSpecial is color.Special: the "special colors" as denoted by xterm.
// These can be set via OSC 5 or via OSC 4 by adding the palette length to
// it.
//
// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
//
// Zig: enum(u3).
type ColorSpecial uint8

const (
	ColorSpecialBold      ColorSpecial = 0
	ColorSpecialUnderline ColorSpecial = 1
	ColorSpecialBlink     ColorSpecial = 2
	ColorSpecialReverse   ColorSpecial = 3
	ColorSpecialItalic    ColorSpecial = 4
)

// Osc4 is color.Special.osc4: the palette length added to the value.
func (self ColorSpecial) Osc4() uint16 {
	// "The special colors can also be set by adding the maximum
	// number of colors (e.g., 88 or 256) to these codes in an
	// OSC 4  control" - xterm ctlseqs
	max := len(ColorPalette{})
	return uint16(self) + uint16(max)
}

// colorSpecialNames are the tag names of ColorSpecial, in order.
var colorSpecialNames = [...]string{
	"bold",
	"underline",
	"blink",
	"reverse",
	"italic",
}

// String returns the Zig tag name (@tagName).
func (self ColorSpecial) String() string { return colorSpecialNames[self] }

// ColorDynamic is color.Dynamic: the "dynamic colors" as denoted by xterm.
// These can be set via OSC 10 through 19.
//
// Zig: enum(u5).
type ColorDynamic uint8

const (
	ColorDynamicForeground          ColorDynamic = 10
	ColorDynamicBackground          ColorDynamic = 11
	ColorDynamicCursor              ColorDynamic = 12
	ColorDynamicPointerForeground   ColorDynamic = 13
	ColorDynamicPointerBackground   ColorDynamic = 14
	ColorDynamicTektronixForeground ColorDynamic = 15
	ColorDynamicTektronixBackground ColorDynamic = 16
	ColorDynamicHighlightBackground ColorDynamic = 17
	ColorDynamicTektronixCursor     ColorDynamic = 18
	ColorDynamicHighlightForeground ColorDynamic = 19
)

// Next is color.Dynamic.next: the next dynamic color sequentially. This is
// required because specifying colors sequentially without their index will
// automatically use the next dynamic color.
//
// "Each successive parameter changes the next color in the list.  The
// value of Ps tells the starting point in the list."
func (self ColorDynamic) Next() Optional[ColorDynamic] {
	// Zig: std.meta.intToEnum(Dynamic, @intFromEnum(self) + 1) catch null.
	// The sum is a u5; self is at most 19, so it cannot overflow.
	next := self + 1
	if next >= ColorDynamicForeground && next <= ColorDynamicHighlightForeground {
		return Some(next)
	}
	return None[ColorDynamic]()
}

// colorDynamicNames are the tag names of ColorDynamic, from 10 (foreground)
// to 19 (highlight_foreground).
var colorDynamicNames = [...]string{
	"foreground",
	"background",
	"cursor",
	"pointer_foreground",
	"pointer_background",
	"tektronix_foreground",
	"tektronix_background",
	"highlight_background",
	"tektronix_cursor",
	"highlight_foreground",
}

// String returns the Zig tag name (@tagName).
func (self ColorDynamic) String() string {
	return colorDynamicNames[self-ColorDynamicForeground]
}

// ColorRGB is color.RGB: RGB.
//
// Zig: packed struct(u24). The packed value is not visible to other code, so
// it is an ordinary Go struct with the same field names.
type ColorRGB struct {
	R uint8
	G uint8
	B uint8
}

// Eql is color.RGB.eql.
func (self ColorRGB) Eql(other ColorRGB) bool {
	return self.R == other.R && self.G == other.G && self.B == other.B
}

// Contrast is color.RGB.contrast: calculates the contrast ratio between two
// colors. The contrast ration is a value between 1 and 21 where 1 is the
// lowest contrast and 21 is the highest contrast.
//
// https://www.w3.org/TR/WCAG20/#contrast-ratiodef
func (self ColorRGB) Contrast(other ColorRGB) float64 {
	// pair[0] = lighter, pair[1] = darker
	pair := func() [2]float64 {
		self_lum := self.Luminance()
		other_lum := other.Luminance()
		if self_lum > other_lum {
			return [2]float64{self_lum, other_lum}
		}
		return [2]float64{other_lum, self_lum}
	}()

	return (pair[0] + 0.05) / (pair[1] + 0.05)
}

// Luminance is color.RGB.luminance: calculates luminance based on the W3C
// formula. This returns a normalized value between 0 and 1 where 0 is black
// and 1 is white.
//
// https://www.w3.org/TR/WCAG20/#relativeluminancedef
func (self ColorRGB) Luminance() float64 {
	r_lum := colorRGBComponentLuminance(self.R)
	g_lum := colorRGBComponentLuminance(self.G)
	b_lum := colorRGBComponentLuminance(self.B)
	return float64(0.2126*r_lum) + float64(0.7152*g_lum) + float64(0.0722*b_lum)
}

// colorRGBComponentLuminance is color.RGB.componentLuminance: calculates
// single-component luminance based on the W3C formula.
//
// Expects sRGB color space which at the time of writing we don't generally
// use but it's a good enough approximation until we fix that.
// https://www.w3.org/TR/WCAG20/#relativeluminancedef
func colorRGBComponentLuminance(c uint8) float64 {
	c_f64 := float64(c)
	normalized := c_f64 / 255
	if normalized <= 0.03928 {
		return normalized / 12.92
	}
	return colorStdMathPowF64((normalized+0.055)/1.055, 2.4)
}

// PerceivedLuminance is color.RGB.perceivedLuminance: calculates "perceived
// luminance" which is better for determining light vs dark.
//
// Source: https://www.w3.org/TR/AERT/#color-contrast
func (self ColorRGB) PerceivedLuminance() float64 {
	r_f64 := float64(self.R)
	g_f64 := float64(self.G)
	b_f64 := float64(self.B)
	return float64(0.299*(r_f64/255)) + float64(0.587*(g_f64/255)) + float64(0.114*(b_f64/255))
}

// colorRGBFromIntensity is color.RGB.fromIntensity: parse a color from a
// floating point intensity value.
//
// The value should be between 0.0 and 1.0, inclusive.
//
// The NaN check is the reference fix of term51 (patches-reference/0002,
// docs/upstream-findings.md section 11): upstream lets NaN through both
// range checks and then aborts in @intFromFloat; the port, like the
// recorded corpus, returns error.InvalidFormat.
func colorRGBFromIntensity(value []uint8) (uint8, error) {
	i, ok := stdFmtParseFloatF64(value)
	if !ok {
		return 0, ErrInvalidFormat
	}
	// term51 reference fix: NaN passes both range checks below and
	// @intFromFloat(NaN) is illegal behaviour (a panic in safe builds).
	if math.IsNaN(i) {
		return 0, ErrInvalidFormat
	}
	if i < 0.0 || i > 1.0 {
		return 0, ErrInvalidFormat
	}

	// Zig: @intFromFloat truncates toward zero; the product is in
	// [-0, 255], so it fits a u8.
	return uint8(i * math.MaxUint8), nil
}

// colorRGBFromHex is color.RGB.fromHex: parse a color from a string of
// hexadecimal digits.
//
// The string can contain 1, 2, 3, or 4 characters and represents the color
// value scaled in 4, 8, 12, or 16 bits, respectively.
func colorRGBFromHex(value []uint8) (uint8, error) {
	if len(value) == 0 || len(value) > 4 {
		return 0, ErrInvalidFormat
	}

	color, ok := colorStdFmtParseUnsigned(value, 16, 16)
	if !ok {
		return 0, ErrInvalidFormat
	}

	var divisor uint
	switch len(value) {
	case 1:
		divisor = 1<<4 - 1 // std.math.maxInt(u4)
	case 2:
		divisor = 1<<8 - 1 // std.math.maxInt(u8)
	case 3:
		divisor = 1<<12 - 1 // std.math.maxInt(u12)
	case 4:
		divisor = 1<<16 - 1 // std.math.maxInt(u16)
	default:
		panic("unreachable")
	}

	// Zig: @intCast to u8, which traps if the value does not fit.
	v := uint(color) * math.MaxUint8 / divisor
	assert(v <= math.MaxUint8)
	return uint8(v), nil
}

// ColorRGBParse is color.RGB.parse: parse a color specification.
//
// Any of the following forms are accepted:
//
// 1. rgb:<red>/<green>/<blue>
//
//	<red>, <green>, <blue> := h | hh | hhh | hhhh
//
//	where `h` is a single hexadecimal digit.
//
// 2. rgbi:<red>/<green>/<blue>
//
//	where <red>, <green>, and <blue> are floating point values between
//	0.0 and 1.0 (inclusive).
//
// 3. #rgb, #rrggbb, #rrrgggbbb #rrrrggggbbbb
//
//	where `r`, `g`, and `b` are a single hexadecimal digit.
//	These specify a color with 4, 8, 12, and 16 bits of precision
//	per color channel.
func ColorRGBParse(value []uint8) (ColorRGB, error) {
	if len(value) == 0 {
		return ColorRGB{}, ErrInvalidFormat
	}

	if value[0] == '#' {
		switch len(value) {
		case 4:
			return colorRGBFromHex3(value[1:2], value[2:3], value[3:4])
		case 7:
			return colorRGBFromHex3(value[1:3], value[3:5], value[5:7])
		case 10:
			return colorRGBFromHex3(value[1:4], value[4:7], value[7:10])
		case 13:
			return colorRGBFromHex3(value[1:5], value[5:9], value[9:13])

		default:
			return ColorRGB{}, ErrInvalidFormat
		}
	}

	// Check for X11 named colors. We allow whitespace around the edges
	// of the color because Kitty allows whitespace. This is not part of
	// any spec I could find.
	if rgb := X11ColorMap.get(colorStdMemTrim(value, " ")); rgb.Valid {
		return rgb.Value, nil
	}

	if len(value) < len("rgb:a/a/a") || string(value[0:3]) != "rgb" {
		return ColorRGB{}, ErrInvalidFormat
	}

	var i uint = 3

	use_intensity := false
	if value[i] == 'i' {
		i += 1
		use_intensity = true
	}

	if value[i] != ':' {
		return ColorRGB{}, ErrInvalidFormat
	}

	i += 1

	var r uint8
	{
		end := colorStdMemIndexOfScalarPos(value, i, '/')
		if !end.Valid {
			return ColorRGB{}, ErrInvalidFormat
		}
		slice := value[i:end.Value]

		i += uint(len(slice)) + 1

		var err error
		if use_intensity {
			r, err = colorRGBFromIntensity(slice)
		} else {
			r, err = colorRGBFromHex(slice)
		}
		if err != nil {
			return ColorRGB{}, err
		}
	}

	var g uint8
	{
		end := colorStdMemIndexOfScalarPos(value, i, '/')
		if !end.Valid {
			return ColorRGB{}, ErrInvalidFormat
		}
		slice := value[i:end.Value]

		i += uint(len(slice)) + 1

		var err error
		if use_intensity {
			g, err = colorRGBFromIntensity(slice)
		} else {
			g, err = colorRGBFromHex(slice)
		}
		if err != nil {
			return ColorRGB{}, err
		}
	}

	var b uint8
	var err error
	if use_intensity {
		b, err = colorRGBFromIntensity(value[i:])
	} else {
		b, err = colorRGBFromHex(value[i:])
	}
	if err != nil {
		return ColorRGB{}, err
	}

	return ColorRGB{
		R: r,
		G: g,
		B: b,
	}, nil
}

// colorRGBFromHex3 is the struct literal `RGB{ .r = try RGB.fromHex(r),
// .g = try RGB.fromHex(g), .b = try RGB.fromHex(b) }` of the `#` forms in
// RGB.parse: the three channels are parsed in order and the first error is
// returned. Not a Zig function.
func colorRGBFromHex3(r []uint8, g []uint8, b []uint8) (ColorRGB, error) {
	rv, err := colorRGBFromHex(r)
	if err != nil {
		return ColorRGB{}, err
	}
	gv, err := colorRGBFromHex(g)
	if err != nil {
		return ColorRGB{}, err
	}
	bv, err := colorRGBFromHex(b)
	if err != nil {
		return ColorRGB{}, err
	}
	return ColorRGB{R: rv, G: gv, B: bv}, nil
}

// colorLAB is color.LAB (private in Zig).
//
// The fields and all arithmetic are f32 in Zig. See the porting note on
// floating point in docs/mapping.md before implementing: Go may fuse a
// multiply and an add into one instruction, which Zig does not, and that
// changes the last bit.
type colorLAB struct {
	l float32
	a float32
	b float32
}

// colorLABFromRgb is color.LAB.fromRgb: RGB to LAB.
func colorLABFromRgb(rgb ColorRGB) colorLAB {
	// Step 1: Normalize sRGB channels from [0, 255] to [0.0, 1.0].
	r := float32(rgb.R) / 255.0
	g := float32(rgb.G) / 255.0
	b := float32(rgb.B) / 255.0

	// Step 2: Apply the inverse sRGB companding (gamma correction) to
	// convert from sRGB to linear RGB. The sRGB transfer function has
	// two segments: a linear portion for small values and a power curve
	// for the rest.
	if r > 0.04045 {
		r = colorStdMathPowF32((r+0.055)/1.055, 2.4)
	} else {
		r = r / 12.92
	}
	if g > 0.04045 {
		g = colorStdMathPowF32((g+0.055)/1.055, 2.4)
	} else {
		g = g / 12.92
	}
	if b > 0.04045 {
		b = colorStdMathPowF32((b+0.055)/1.055, 2.4)
	} else {
		b = b / 12.92
	}

	// Step 3: Convert linear RGB to CIE XYZ using the sRGB to XYZ
	// transformation matrix (D65 illuminant). The X and Z values are
	// normalized by the D65 white point reference values (Xn=0.95047,
	// Zn=1.08883; Yn=1.0 is implicit).
	x := (float32(r*0.4124564) + float32(g*0.3575761) + float32(b*0.1804375)) / 0.95047
	y := float32(r*0.2126729) + float32(g*0.7151522) + float32(b*0.0721750)
	z := (float32(r*0.0193339) + float32(g*0.1191920) + float32(b*0.9503041)) / 1.08883

	// Step 4: Apply the CIE f(t) nonlinear transform to each XYZ
	// component. Above the threshold (epsilon ≈ 0.008856) the cube
	// root is used; below it, a linear approximation avoids numerical
	// instability near zero.
	if x > 0.008856 {
		x = colorStdMathCbrtF32(x)
	} else {
		x = float32(7.787*x) + 16.0/116.0
	}
	if y > 0.008856 {
		y = colorStdMathCbrtF32(y)
	} else {
		y = float32(7.787*y) + 16.0/116.0
	}
	if z > 0.008856 {
		z = colorStdMathCbrtF32(z)
	} else {
		z = float32(7.787*z) + 16.0/116.0
	}

	// Step 5: Compute the final CIELAB values from the transformed XYZ.
	// L* is lightness (0–100), a* is green–red, b* is blue–yellow.
	return colorLAB{l: float32(116.0*y) - 16.0, a: 500.0 * (x - y), b: 200.0 * (y - z)}
}

// toRgb is color.LAB.toRgb: LAB to RGB.
func (self colorLAB) toRgb() ColorRGB {
	// Step 1: Recover the intermediate f(Y), f(X), f(Z) values from
	// L*a*b* by inverting the CIELAB formulas.
	y := (self.l + 16.0) / 116.0
	x := self.a/500.0 + y
	z := y - self.b/200.0

	// Step 2: Apply the inverse CIE f(t) transform to get back to
	// XYZ. Above epsilon (≈0.008856) the cube is used; below it the
	// linear segment is inverted. Results are then scaled by the D65
	// white point reference values (Xn=0.95047, Zn=1.08883; Yn=1.0).
	x3 := x * x * x
	y3 := y * y * y
	z3 := z * z * z
	var xf, yf, zf float32
	if x3 > 0.008856 {
		xf = x3 * 0.95047
	} else {
		xf = (x - 16.0/116.0) / 7.787 * 0.95047
	}
	if y3 > 0.008856 {
		yf = y3
	} else {
		yf = (y - 16.0/116.0) / 7.787
	}
	if z3 > 0.008856 {
		zf = z3 * 1.08883
	} else {
		zf = (z - 16.0/116.0) / 7.787 * 1.08883
	}

	// Step 3: Convert CIE XYZ back to linear RGB using the XYZ to sRGB
	// matrix (inverse of the sRGB to XYZ matrix, D65 illuminant).
	r := float32(xf*3.2404542) - float32(yf*1.5371385) - float32(zf*0.4985314)
	g := float32(-xf*0.9692660) + float32(yf*1.8760108) + float32(zf*0.0415560)
	b := float32(xf*0.0556434) - float32(yf*0.2040259) + float32(zf*1.0572252)

	// Step 4: Apply sRGB companding (gamma correction) to convert from
	// linear RGB back to sRGB. This is the forward sRGB transfer
	// function with the same two-segment split as the inverse.
	if r > 0.0031308 {
		r = float32(1.055*colorStdMathPowF32(r, 1.0/2.4)) - 0.055
	} else {
		r = 12.92 * r
	}
	if g > 0.0031308 {
		g = float32(1.055*colorStdMathPowF32(g, 1.0/2.4)) - 0.055
	} else {
		g = 12.92 * g
	}
	if b > 0.0031308 {
		b = float32(1.055*colorStdMathPowF32(b, 1.0/2.4)) - 0.055
	} else {
		b = 12.92 * b
	}

	// Step 5: Clamp to [0.0, 1.0], scale to [0, 255], and round to
	// the nearest integer to produce the final 8-bit sRGB values.
	return ColorRGB{
		R: colorLABChannel(r),
		G: colorLABChannel(g),
		B: colorLABChannel(b),
	}
}

// colorLABChannel is `@intFromFloat(@min(@max(c, 0.0), 1.0) * 255.0 + 0.5)`
// of LAB.toRgb. Not a Zig function. Zig's @max and @min return the other
// operand when one is NaN (Go's built-in max and min return NaN), so they
// are colorZigMaxF32 and colorZigMinF32; @intFromFloat truncates, and the value is in
// [0.5, 255.5], which fits a u8.
func colorLABChannel(c float32) uint8 {
	return uint8(float32(colorZigMinF32(colorZigMaxF32(c, 0.0), 1.0)*255.0) + 0.5)
}

// colorLABLerp is color.LAB.lerp: linearly interpolate between two LAB
// colors component-wise. `t` is the interpolation factor (0 = a, 1 = b).
func colorLABLerp(t float32, a colorLAB, b colorLAB) colorLAB {
	return colorLAB{
		l: a.l + float32(t*(b.l-a.l)),
		a: a.a + float32(t*(b.a-a.a)),
		b: a.b + float32(t*(b.b-a.b)),
	}
}

// ---------------------------------------------------------------------------
// Zig standard-library and compiler_rt stand-ins used by this file and by
// x11_color.go and kitty_color.go. Nothing here is ported from a Ghostty
// source file; each one follows the Zig 0.15.2 source it names (the
// toolchain that built the recorded corpus), because the Go standard
// library differs (see docs/mapping.md, "Colours and SGR", "Floating point
// (all of `color.zig`)"). They belong in
// zig.go; they are here so that this port adds no code to a file another
// port is editing.
//
// Every product that is an operand of an addition or subtraction is wrapped
// in an explicit conversion so that Go cannot fuse the two (see the header
// of this file).
//
// std.fmt.parseFloat is not here: it is in std_fmt_parse_float.go (which
// uses colorStdMathNanF64 for std.math.nan(f64)).

// colorStdMemTrim is std.mem.trim(u8, slice, values_to_strip): slice
// without the leading and trailing bytes that are in values_to_strip.
func colorStdMemTrim(slice []uint8, values_to_strip string) []uint8 {
	in := func(c uint8) bool {
		for i := 0; i < len(values_to_strip); i++ {
			if values_to_strip[i] == c {
				return true
			}
		}
		return false
	}
	begin := 0
	end := len(slice)
	for begin < end && in(slice[begin]) {
		begin += 1
	}
	for end > begin && in(slice[end-1]) {
		end -= 1
	}
	return slice[begin:end]
}

// colorStdMemIndexOfScalarPos is std.mem.indexOfScalarPos(u8, slice,
// start_index, value): the index of the first value at or after
// start_index, or null (also when start_index >= slice.len).
func colorStdMemIndexOfScalarPos(slice []uint8, start_index uint, value uint8) Optional[uint] {
	if start_index >= uint(len(slice)) {
		return None[uint]()
	}
	for i := start_index; i < uint(len(slice)); i++ {
		if slice[i] == value {
			return Some(i)
		}
	}
	return None[uint]()
}

// colorStdFmtCharToDigit is std.fmt.charToDigit.
func colorStdFmtCharToDigit(c uint8, base uint8) (uint8, bool) {
	var value uint8
	switch {
	case c >= '0' && c <= '9':
		value = c - '0'
	case c >= 'A' && c <= 'Z':
		value = c - 'A' + 10
	case c >= 'a' && c <= 'z':
		value = c - 'a' + 10
	default:
		return 0, false
	}

	if value >= base {
		return 0, false
	}

	return value, true
}

// colorStdFmtParseUnsigned is std.fmt.parseUnsigned(T, buf, base) for an
// unsigned T of `bits` bits (8 or 16 here), that is parseIntWithSign with
// sign .pos: no sign is accepted, '_' is skipped between digits but may not
// be first or last, and the digits accumulate in an integer of
// max(8, bits) bits, so that an overflow is an error.
//
// Zig returns error.InvalidCharacter or error.Overflow; every caller in the
// port discards which (`catch`), so the Go function reports only success.
// Base 0 (prefix detection) is not used by any caller and is not ported.
func colorStdFmtParseUnsigned(buf []uint8, base uint8, bits uint) (uint64, bool) {
	assert(base >= 2 && base <= 36)
	assert(bits >= 1 && bits <= 16)
	if len(buf) == 0 {
		return 0, false
	}

	// accumulate into Accumulate which is always 8 bits or larger.  this prevents
	// `buf_base` from overflowing Result.
	acc_limit := uint64(1)<<max(8, bits) - 1
	var accumulate uint64 = 0

	if buf[0] == '_' || buf[len(buf)-1] == '_' {
		return 0, false
	}

	for _, c := range buf {
		if c == '_' {
			continue
		}
		digit, ok := colorStdFmtCharToDigit(c, base)
		if !ok {
			return 0, false
		}
		if accumulate != 0 {
			// Zig: math.mul, error.Overflow when the product does not fit.
			accumulate = accumulate * uint64(base)
			if accumulate > acc_limit {
				return 0, false
			}
		}
		// Zig: math.add, error.Overflow when the sum does not fit.
		accumulate = accumulate + uint64(digit)
		if accumulate > acc_limit {
			return 0, false
		}
	}

	// Zig: math.cast(Result, accumulate) orelse error.Overflow.
	if accumulate > uint64(1)<<bits-1 {
		return 0, false
	}
	return accumulate, true
}

// colorStdMathNanF64 is std.math.nan(f64), the quiet NaN 0x7FF8000000000000
// (Go's math.NaN() has a different payload).
func colorStdMathNanF64() float64 {
	return math.Float64frombits(0x7FF8000000000000)
}

// colorStdMathNanF32 is std.math.nan(f32), the quiet NaN 0x7FC00000.
func colorStdMathNanF32() float32 {
	return math.Float32frombits(0x7FC00000)
}

// colorZigMaxF32 is Zig's @max on f32: when one operand is NaN the other is
// returned (LLVM maxnum). Go's built-in max returns NaN.
func colorZigMaxF32(a float32, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if a > b {
		return a
	}
	return b
}

// colorZigMinF32 is Zig's @min on f32: when one operand is NaN the other is
// returned (LLVM minnum). Go's built-in min returns NaN.
func colorZigMinF32(a float32, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if a < b {
		return a
	}
	return b
}

// colorStdMathIsOddInteger is std/math/pow.zig isOddInteger.
func colorStdMathIsOddInteger(x float64) bool {
	if math.Abs(x) >= 1<<53 {
		return false
	}
	ipart := math.Trunc(x)
	fpart := x - ipart
	return fpart == 0.0 && int64(ipart)&1 == 1
}

// colorStdMathPowF32 is std.math.pow(f32, x, y) (std/math/pow.zig, itself
// ported from Go's math.Pow but computed in f32, with @exp and @log the
// compiler_rt functions expf and logf).
func colorStdMathPowF32(x float32, y float32) float32 {
	if y == 0 || x == 1 {
		return 1
	}

	if x != x || y != y {
		return colorStdMathNanF32()
	}

	if y == 1 {
		return x
	}

	if x == 0 {
		if y < 0 {
			// pow(+-0, y) = +-inf for y an odd integer
			if colorStdMathIsOddInteger(float64(y)) {
				return float32(math.Copysign(math.Inf(1), float64(x)))
			} else {
				// pow(+-0, y) = +inf for y an even integer
				return float32(math.Inf(1))
			}
		} else {
			if colorStdMathIsOddInteger(float64(y)) {
				return x
			} else {
				return 0
			}
		}
	}

	if math.IsInf(float64(y), 0) {
		// pow(-1, +-inf) = 1
		if x == -1 {
			return 1.0
		} else if (float32(math.Abs(float64(x))) < 1) == math.IsInf(float64(y), 1) {
			// pow(x, +inf) = +0    for |x| < 1
			// pow(x, -inf) = +0    for |x| > 1
			return 0
		} else {
			// pow(x, +inf) = +inf  for |x| > 1
			// pow(x, -inf) = +inf  for |x| < 1
			return float32(math.Inf(1))
		}
	}

	if math.IsInf(float64(x), 0) {
		if math.IsInf(float64(x), -1) {
			return colorStdMathPowF32(1/x, -y)
		} else if y < 0 {
			// pow(+inf, y) = +0 for y < 0
			return 0
		} else if y > 0 {
			// pow(+inf, y) = +0 for y > 0
			return float32(math.Inf(1))
		}
	}

	// special case sqrt
	if y == 0.5 {
		// Zig: @sqrt on f32. The square root rounded to f64 and then to f32
		// is the correctly rounded f32 square root.
		return float32(math.Sqrt(float64(x)))
	}

	if y == -0.5 {
		return 1 / float32(math.Sqrt(float64(x)))
	}

	// Zig: math.modf(@abs(y)); @trunc is exact in Go's float64.
	abs_y := float32(math.Abs(float64(y)))
	yi := float32(math.Trunc(float64(abs_y)))
	yf := abs_y - yi

	// pow(x, y) = nan for x < 0 and y not an integer
	if yf != 0 && x < 0 {
		return colorStdMathNanF32()
	}
	if yi >= 1<<31 {
		return colorCompilerRtExpf(y * colorCompilerRtLogf(x))
	}

	// a = a1 * 2^ae
	var a1 float32 = 1.0
	var ae int32 = 0

	// a *= x^yf
	if yf != 0 {
		if yf > 0.5 {
			yf -= 1
			yi += 1
		}
		a1 = colorCompilerRtExpf(yf * colorCompilerRtLogf(x))
	}

	// ans = a1 * 2^ae
	// Zig: math.frexp(x). The f32 value is exact in float64, so Go's Frexp
	// gives the same significand and exponent.
	x1_64, xe_int := math.Frexp(float64(x))
	xe := int32(xe_int)
	x1 := float32(x1_64)

	// a *= x^yi
	i := int32(yi)
	for i != 0 {
		const overflow_shift = 8 + 1 // math.floatExponentBits(f32) + 1
		if xe < -(1<<overflow_shift) || (1<<overflow_shift) < xe {
			// catastrophic overflow partial-cancellation
			ae += xe
			break
		}
		if i&1 == 1 {
			a1 *= x1
			ae += xe
		}
		x1 = float32(x1 * x1)
		xe <<= 1
		if x1 < 0.5 {
			x1 += x1
			xe -= 1
		}
		i >>= 1
	}

	// a *= a1 * 2^ae
	if y < 0 {
		a1 = 1 / a1
		ae = -ae
	}

	return colorStdMathLdexpF32(a1, ae)
}

// colorStdMathPowF64 is std.math.pow(f64, x, y): colorStdMathPowF32 in
// f64, with @exp and @log the compiler_rt functions exp and log.
func colorStdMathPowF64(x float64, y float64) float64 {
	if y == 0 || x == 1 {
		return 1
	}

	if x != x || y != y {
		return colorStdMathNanF64()
	}

	if y == 1 {
		return x
	}

	if x == 0 {
		if y < 0 {
			// pow(+-0, y) = +-inf for y an odd integer
			if colorStdMathIsOddInteger(y) {
				return math.Copysign(math.Inf(1), x)
			} else {
				// pow(+-0, y) = +inf for y an even integer
				return math.Inf(1)
			}
		} else {
			if colorStdMathIsOddInteger(y) {
				return x
			} else {
				return 0
			}
		}
	}

	if math.IsInf(y, 0) {
		// pow(-1, +-inf) = 1
		if x == -1 {
			return 1.0
		} else if (math.Abs(x) < 1) == math.IsInf(y, 1) {
			// pow(x, +inf) = +0    for |x| < 1
			// pow(x, -inf) = +0    for |x| > 1
			return 0
		} else {
			// pow(x, +inf) = +inf  for |x| > 1
			// pow(x, -inf) = +inf  for |x| < 1
			return math.Inf(1)
		}
	}

	if math.IsInf(x, 0) {
		if math.IsInf(x, -1) {
			return colorStdMathPowF64(1/x, -y)
		} else if y < 0 {
			// pow(+inf, y) = +0 for y < 0
			return 0
		} else if y > 0 {
			// pow(+inf, y) = +0 for y > 0
			return math.Inf(1)
		}
	}

	// special case sqrt
	if y == 0.5 {
		return math.Sqrt(x)
	}

	if y == -0.5 {
		return 1 / math.Sqrt(x)
	}

	abs_y := math.Abs(y)
	yi := math.Trunc(abs_y)
	yf := abs_y - yi

	// pow(x, y) = nan for x < 0 and y not an integer
	if yf != 0 && x < 0 {
		return colorStdMathNanF64()
	}
	if yi >= 1<<63 {
		return colorCompilerRtExp(y * colorCompilerRtLog(x))
	}

	// a = a1 * 2^ae
	var a1 float64 = 1.0
	var ae int32 = 0

	// a *= x^yf
	if yf != 0 {
		if yf > 0.5 {
			yf -= 1
			yi += 1
		}
		a1 = colorCompilerRtExp(yf * colorCompilerRtLog(x))
	}

	// ans = a1 * 2^ae
	x1, xe_int := math.Frexp(x)
	xe := int32(xe_int)

	// a *= x^yi
	i := int64(yi)
	for i != 0 {
		const overflow_shift = 11 + 1 // math.floatExponentBits(f64) + 1
		if xe < -(1<<overflow_shift) || (1<<overflow_shift) < xe {
			// catastrophic overflow partial-cancellation
			ae += xe
			break
		}
		if i&1 == 1 {
			a1 *= x1
			ae += xe
		}
		x1 = float64(x1 * x1)
		xe <<= 1
		if x1 < 0.5 {
			x1 += x1
			xe -= 1
		}
		i >>= 1
	}

	// a *= a1 * 2^ae
	if y < 0 {
		a1 = 1 / a1
		ae = -ae
	}

	return colorStdMathLdexpF64(a1, ae)
}

// colorStdMathLdexpF32 is std.math.scalbn (std.math.ldexp) on f32: x * 2^n,
// with Zig's rounding of a subnormal result.
func colorStdMathLdexpF32(x float32, n int32) float32 {
	const exponent_bits = 8
	const mantissa_bits = 23
	const fractional_bits = 23
	const max_biased_exponent = 2 * 127
	const mantissa_mask = uint32(1)<<mantissa_bits - 1

	repr := math.Float32bits(x)
	sign_bit := repr & (1 << (exponent_bits + mantissa_bits))

	if x != x || math.IsInf(float64(x), 0) {
		return x
	}

	exponent := int32((repr << 1) >> (mantissa_bits + 1))
	if exponent == 0 {
		exponent += exponent_bits - int32(bits.LeadingZeros32(repr<<1))
	}

	if n >= 0 {
		if n > max_biased_exponent-exponent {
			return math.Float32frombits(math.Float32bits(float32(math.Inf(1))) | sign_bit)
		} else if exponent+n <= 0 {
			return math.Float32frombits((repr << uint(n)) | sign_bit)
		} else if exponent <= 0 {
			result := uint32(n+exponent) << mantissa_bits
			result |= (repr << uint(1-exponent)) & mantissa_mask
			return math.Float32frombits(result | sign_bit)
		}

		return math.Float32frombits(repr + (uint32(n) << mantissa_bits))
	} else {
		if n <= -exponent {
			if n < -(mantissa_bits + exponent) {
				return math.Float32frombits(sign_bit) // Severe underflow. Return +/- 0
			}

			shift := uint(min(-n, -(exponent+n)+1))
			exact_tie := uint(bits.TrailingZeros32(repr)) == shift-1
			result := repr & mantissa_mask

			// Include integer bit
			if exponent > 0 {
				result |= 1 << fractional_bits
			}
			result = result >> (shift - 1)

			var tie uint32
			if exact_tie {
				tie = 1
			}
			result = ((result + 1) >> 1) &^ tie
			return math.Float32frombits(result | sign_bit)
		}

		return math.Float32frombits(repr - (uint32(-n) << mantissa_bits))
	}
}

// colorStdMathLdexpF64 is std.math.scalbn (std.math.ldexp) on f64.
func colorStdMathLdexpF64(x float64, n int32) float64 {
	const exponent_bits = 11
	const mantissa_bits = 52
	const fractional_bits = 52
	const max_biased_exponent = 2 * 1023
	const mantissa_mask = uint64(1)<<mantissa_bits - 1

	repr := math.Float64bits(x)
	sign_bit := repr & (1 << (exponent_bits + mantissa_bits))

	if x != x || math.IsInf(x, 0) {
		return x
	}

	exponent := int32((repr << 1) >> (mantissa_bits + 1))
	if exponent == 0 {
		exponent += exponent_bits - int32(bits.LeadingZeros64(repr<<1))
	}

	if n >= 0 {
		if n > max_biased_exponent-exponent {
			return math.Float64frombits(math.Float64bits(math.Inf(1)) | sign_bit)
		} else if exponent+n <= 0 {
			return math.Float64frombits((repr << uint(n)) | sign_bit)
		} else if exponent <= 0 {
			result := uint64(n+exponent) << mantissa_bits
			result |= (repr << uint(1-exponent)) & mantissa_mask
			return math.Float64frombits(result | sign_bit)
		}

		return math.Float64frombits(repr + (uint64(n) << mantissa_bits))
	} else {
		if n <= -exponent {
			if n < -(mantissa_bits + exponent) {
				return math.Float64frombits(sign_bit) // Severe underflow. Return +/- 0
			}

			shift := uint(min(-n, -(exponent+n)+1))
			exact_tie := uint(bits.TrailingZeros64(repr)) == shift-1
			result := repr & mantissa_mask

			// Include integer bit
			if exponent > 0 {
				result |= 1 << fractional_bits
			}
			result = result >> (shift - 1)

			var tie uint64
			if exact_tie {
				tie = 1
			}
			result = ((result + 1) >> 1) &^ tie
			return math.Float64frombits(result | sign_bit)
		}

		return math.Float64frombits(repr - (uint64(-n) << mantissa_bits))
	}
}

// colorStdMathCbrtF32 is std.math.cbrt on f32 (std/math/cbrt.zig cbrt32,
// ported from musl): returns the cube root of x.
func colorStdMathCbrtF32(x float32) float32 {
	const B1 uint32 = 709958130 // (127 - 127.0 / 3 - 0.03306235651) * 2^23
	const B2 uint32 = 642849266 // (127 - 127.0 / 3 - 24 / 3 - 0.03306235651) * 2^23

	u := math.Float32bits(x)
	hx := u & 0x7FFFFFFF

	// cbrt(nan, inf) = itself
	if hx >= 0x7F800000 {
		return x + x
	}

	// cbrt to ~5bits
	if hx < 0x00800000 {
		// cbrt(+-0) = itself
		if hx == 0 {
			return x
		}
		u = math.Float32bits(x * 0x1.0p24)
		hx = u & 0x7FFFFFFF
		hx = hx/3 + B2
	} else {
		hx = hx/3 + B1
	}

	u &= 0x80000000
	u |= hx

	// first step newton to 16 bits
	var t float64 = float64(math.Float32frombits(u))
	var r float64 = float64(t * t * t)
	t = t * (float64(x) + float64(x) + r) / (float64(x) + r + r)

	// second step newton to 47 bits
	r = float64(t * t * t)
	t = t * (float64(x) + float64(x) + r) / (float64(x) + r + r)

	return float32(t)
}

// colorCompilerRtExpf is compiler_rt expf (lib/compiler_rt/exp.zig, ported
// from musl), the function Zig's @exp calls on f32.
func colorCompilerRtExpf(x_ float32) float32 {
	half := [2]float32{0.5, -0.5}
	const ln2hi float32 = 6.9314575195e-1
	const ln2lo float32 = 1.4286067653e-6
	const invln2 float32 = 1.4426950216e+0
	const P1 float32 = 1.6666625440e-1
	const P2 float32 = -2.7667332906e-3

	x := x_
	hx := math.Float32bits(x)
	sign := int32(hx >> 31)
	hx &= 0x7FFFFFFF

	if x != x {
		return x
	}

	// |x| >= -87.33655 or nan
	if hx >= 0x42AEAC50 {
		// nan
		if hx > 0x7F800000 {
			return x
		}
		// x >= 88.722839
		if hx >= 0x42b17218 && sign == 0 {
			return x * 0x1.0p127
		}
		if sign != 0 {
			// x <= -103.972084
			if hx >= 0x42CFF1B5 {
				return 0
			}
		}
	}

	var k int32
	var hi float32
	var lo float32

	// |x| > 0.5 * ln2
	if hx > 0x3EB17218 {
		// |x| > 1.5 * ln2
		if hx > 0x3F851592 {
			// Zig: @intFromFloat truncates toward zero.
			k = int32(float32(invln2*x) + half[sign])
		} else {
			k = 1 - sign - sign
		}

		fk := float32(k)
		hi = x - float32(fk*ln2hi)
		lo = float32(fk * ln2lo)
		x = hi - lo
	} else if hx > 0x39000000 {
		// |x| > 2^(-14)
		k = 0
		hi = x
		lo = 0
	} else {
		return 1 + x
	}

	xx := x * x
	c := x - float32(xx*(P1+float32(xx*P2)))
	y := 1 + (x*c/(2-c) - lo + hi)

	if k == 0 {
		return y
	} else {
		return colorStdMathLdexpF32(y, k)
	}
}

// colorCompilerRtLogf is compiler_rt logf (lib/compiler_rt/log.zig, ported
// from musl), the function Zig's @log calls on f32.
func colorCompilerRtLogf(x_ float32) float32 {
	const ln2_hi float32 = 6.9313812256e-01
	const ln2_lo float32 = 9.0580006145e-06
	const Lg1 float32 = 0xaaaaaa.0p-24
	const Lg2 float32 = 0xccce13.0p-25
	const Lg3 float32 = 0x91e9ee.0p-25
	const Lg4 float32 = 0xf89e26.0p-26

	x := x_
	ix := math.Float32bits(x)
	var k int32 = 0

	// x < 2^(-126)
	if ix < 0x00800000 || ix>>31 != 0 {
		// log(+-0) = -inf
		if ix<<1 == 0 {
			return float32(math.Inf(-1))
		}
		// log(-#) = nan
		if ix>>31 != 0 {
			return colorStdMathNanF32()
		}

		// subnormal, scale x
		k -= 25
		x *= 0x1.0p25
		ix = math.Float32bits(x)
	} else if ix >= 0x7F800000 {
		return x
	} else if ix == 0x3F800000 {
		return 0
	}

	// x into [sqrt(2) / 2, sqrt(2)]
	ix += 0x3F800000 - 0x3F3504F3
	k += int32(ix>>23) - 0x7F
	ix = (ix & 0x007FFFFF) + 0x3F3504F3
	x = math.Float32frombits(ix)

	f := x - 1.0
	s := f / (2.0 + f)
	z := float32(s * s)
	w := float32(z * z)
	t1 := float32(w * (Lg2 + float32(w*Lg4)))
	t2 := float32(z * (Lg1 + float32(w*Lg3)))
	R := t2 + t1
	hfsq := float32(0.5 * f * f)
	dk := float32(k)

	return float32(s*(hfsq+R)) + float32(dk*ln2_lo) - hfsq + f + float32(dk*ln2_hi)
}

// colorCompilerRtExp is compiler_rt exp (lib/compiler_rt/exp.zig, ported
// from musl), the function Zig's @exp calls on f64.
func colorCompilerRtExp(x_ float64) float64 {
	half := [2]float64{0.5, -0.5}
	const ln2hi float64 = 6.93147180369123816490e-01
	const ln2lo float64 = 1.90821492927058770002e-10
	const invln2 float64 = 1.44269504088896338700e+00
	const P1 float64 = 1.66666666666666019037e-01
	const P2 float64 = -2.77777777770155933842e-03
	const P3 float64 = 6.61375632143793436117e-05
	const P4 float64 = -1.65339022054652515390e-06
	const P5 float64 = 4.13813679705723846039e-08

	x := x_
	ux := math.Float64bits(x)
	hx := ux >> 32
	sign := int32(hx >> 31)
	hx &= 0x7FFFFFFF

	if x != x {
		return x
	}

	// |x| >= 708.39 or nan
	if hx >= 0x4086232B {
		// nan
		if hx > 0x7FF00000 {
			return x
		}
		if x > 709.782712893383973096 {
			// overflow if x != inf
			return math.Inf(1)
		}
		if x < -708.39641853226410622 {
			// underflow if x != -inf
			if x < -745.13321910194110842 {
				return 0
			}
		}
	}

	// argument reduction
	var k int32
	var hi float64
	var lo float64

	// |x| > 0.5 * ln2
	if hx > 0x3FD62E42 {
		// |x| >= 1.5 * ln2
		if hx > 0x3FF0A2B2 {
			// Zig: @intFromFloat truncates toward zero.
			k = int32(float64(invln2*x) + half[sign])
		} else {
			k = 1 - sign - sign
		}

		dk := float64(k)
		hi = x - float64(dk*ln2hi)
		lo = float64(dk * ln2lo)
		x = hi - lo
	} else if hx > 0x3E300000 {
		// |x| > 2^(-28)
		k = 0
		hi = x
		lo = 0
	} else {
		// inexact if x != 0
		return 1 + x
	}

	xx := x * x
	c := x - float64(xx*(P1+float64(xx*(P2+float64(xx*(P3+float64(xx*(P4+float64(xx*P5)))))))))
	y := 1 + (x*c/(2-c) - lo + hi)

	if k == 0 {
		return y
	} else {
		return colorStdMathLdexpF64(y, k)
	}
}

// colorCompilerRtLog is compiler_rt log (lib/compiler_rt/log.zig, ported
// from musl), the function Zig's @log calls on f64.
func colorCompilerRtLog(x_ float64) float64 {
	const ln2_hi float64 = 6.93147180369123816490e-01
	const ln2_lo float64 = 1.90821492927058770002e-10
	const Lg1 float64 = 6.666666666666735130e-01
	const Lg2 float64 = 3.999999999940941908e-01
	const Lg3 float64 = 2.857142874366239149e-01
	const Lg4 float64 = 2.222219843214978396e-01
	const Lg5 float64 = 1.818357216161805012e-01
	const Lg6 float64 = 1.531383769920937332e-01
	const Lg7 float64 = 1.479819860511658591e-01

	x := x_
	ix := math.Float64bits(x)
	hx := uint32(ix >> 32)
	var k int32 = 0

	if hx < 0x00100000 || hx>>31 != 0 {
		// log(+-0) = -inf
		if ix<<1 == 0 {
			return math.Inf(-1)
		}
		// log(-#) = nan
		if hx>>31 != 0 {
			return colorStdMathNanF64()
		}

		// subnormal, scale x
		k -= 54
		x *= 0x1p54
		hx = uint32(math.Float64bits(x) >> 32)
	} else if hx >= 0x7FF00000 {
		return x
	} else if hx == 0x3FF00000 && ix<<32 == 0 {
		return 0
	}

	// x into [sqrt(2) / 2, sqrt(2)]
	hx += 0x3FF00000 - 0x3FE6A09E
	k += int32(hx>>20) - 0x3FF
	hx = (hx & 0x000FFFFF) + 0x3FE6A09E
	ix = (uint64(hx) << 32) | (ix & 0xFFFFFFFF)
	x = math.Float64frombits(ix)

	f := x - 1.0
	hfsq := float64(0.5 * f * f)
	s := f / (2.0 + f)
	z := float64(s * s)
	w := float64(z * z)
	t1 := float64(w * (Lg2 + float64(w*(Lg4+float64(w*Lg6)))))
	t2 := float64(z * (Lg1 + float64(w*(Lg3+float64(w*(Lg5+float64(w*Lg7)))))))
	R := t2 + t1
	dk := float64(k)

	return float64(s*(hfsq+R)) + float64(dk*ln2_lo) - hfsq + f + float64(dk*ln2_hi)
}
