// WIREFRAME for src/terminal/color.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then.
//
// Declarations appear in the same order as in color.zig. The C ABI
// declarations (`C`, `cval`) are not ported; see docs/mapping.md.

package terminal

import "errors"

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
// the 6x6x6 cube, then the grayscale ramp. Wireframe: left zero; the
// implementation initialises it once at start-up.
var ColorDefault ColorPalette

// ColorPalette is color.Palette: palette is the 256 color palette.
type ColorPalette [256]ColorRGB

// ColorPaletteMask is color.PaletteMask: mask that can be used to set which
// palette indexes were set.
//
// Zig: std.StaticBitSet(256).
type ColorPaletteMask = StaticBitSet256

// StaticBitSet256 is std.StaticBitSet(256). It belongs in zig.go beside
// StaticBitSet24 once implemented; it is declared here so that the wireframe
// adds no code to an existing file. Methods keep Zig's names.
type StaticBitSet256 struct {
	masks [4]uint64
}

// StaticBitSet256BitLength is std.StaticBitSet(256).bit_length.
const StaticBitSet256BitLength = 256

// StaticBitSet256InitEmpty is std.StaticBitSet(256).initEmpty.
func StaticBitSet256InitEmpty() StaticBitSet256 { panic(notImplemented) }

// StaticBitSet256InitFull is std.StaticBitSet(256).initFull.
func StaticBitSet256InitFull() StaticBitSet256 { panic(notImplemented) }

// isSet reports whether the bit at index is set. Zig asserts
// index < bit_length.
func (self StaticBitSet256) isSet(index int) bool { panic(notImplemented) }

// set sets the bit at index. Zig asserts index < bit_length.
func (self *StaticBitSet256) set(index int) { panic(notImplemented) }

// unset clears the bit at index. Zig asserts index < bit_length.
func (self *StaticBitSet256) unset(index int) { panic(notImplemented) }

// count returns the number of set bits.
func (self StaticBitSet256) count() int { panic(notImplemented) }

// iterator returns an iterator over the indexes of the set bits, in
// ascending order. Zig: iterator(.{}).
func (self *StaticBitSet256) iterator() StaticBitSet256Iterator { panic(notImplemented) }

// StaticBitSet256Iterator is the iterator returned by
// StaticBitSet256.iterator.
type StaticBitSet256Iterator struct {
	set  *StaticBitSet256
	next int
}

// Next returns the next set index, or none at the end. Zig: next() ?usize.
func (self *StaticBitSet256Iterator) Next() Optional[int] { panic(notImplemented) }

// ColorGenerate256Color is color.generate256Color: generate the 256-color
// palette from the user's base16 theme colors, terminal background, and
// terminal foreground. See color.zig for the full description.
//
// Fill `skip` with user-defined color indexes to avoid replacing them.
func ColorGenerate256Color(
	base ColorPalette,
	skip ColorPaletteMask,
	bg ColorRGB,
	fg ColorRGB,
	harmonious bool,
) ColorPalette {
	panic(notImplemented)
}

// ColorDynamicPalette is color.DynamicPalette: a palette that can have its
// colors changed and reset. Purposely built for terminal color operations.
type ColorDynamicPalette struct {
	// The current palette including any user modifications.
	current ColorPalette

	// The original/default palette values.
	original ColorPalette

	// A bitset where each bit represents whether the corresponding
	// palette index has been modified from its default value.
	mask ColorPaletteMask
}

// ColorDynamicPaletteDefault is color.DynamicPalette.default:
// init(color.default). Wireframe: left zero.
var ColorDynamicPaletteDefault ColorDynamicPalette

// ColorDynamicPaletteInit is color.DynamicPalette.init: initialize a dynamic
// palette with a default palette.
func ColorDynamicPaletteInit(def ColorPalette) ColorDynamicPalette { panic(notImplemented) }

// Set is color.DynamicPalette.set: set a custom color at the given palette
// index.
func (self *ColorDynamicPalette) Set(idx uint8, color ColorRGB) { panic(notImplemented) }

// Reset is color.DynamicPalette.reset: reset the color at the given palette
// index to its original value.
func (self *ColorDynamicPalette) Reset(idx uint8) { panic(notImplemented) }

// ResetAll is color.DynamicPalette.resetAll: reset all colors to their
// original values.
func (self *ColorDynamicPalette) ResetAll() { panic(notImplemented) }

// ChangeDefault is color.DynamicPalette.changeDefault: change the default
// palette, but preserve the changed values.
func (self *ColorDynamicPalette) ChangeDefault(def ColorPalette) { panic(notImplemented) }

// ColorDynamicRGB is color.DynamicRGB: RGB value that can be changed and
// reset. This can also be totally unset in every way, in which case the
// caller can determine their own ultimate default.
type ColorDynamicRGB struct {
	override Optional[ColorRGB]
	default_ Optional[ColorRGB]
}

// ColorDynamicRGBUnset is color.DynamicRGB.unset: both fields null. The zero
// value is already that state.
var ColorDynamicRGBUnset = ColorDynamicRGB{}

// ColorDynamicRGBInit is color.DynamicRGB.init.
func ColorDynamicRGBInit(def ColorRGB) ColorDynamicRGB { panic(notImplemented) }

// Get is color.DynamicRGB.get.
func (self *ColorDynamicRGB) Get() Optional[ColorRGB] { panic(notImplemented) }

// Set is color.DynamicRGB.set.
func (self *ColorDynamicRGB) Set(color ColorRGB) { panic(notImplemented) }

// Reset is color.DynamicRGB.reset.
func (self *ColorDynamicRGB) Reset() { panic(notImplemented) }

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

// String returns the Zig tag name (@tagName).
//
// Unnamed values (16 to 255) return ""; dumps print them as #N.
func (self ColorName) String() string { panic(notImplemented) }

// Default is color.Name.default: default colors for tagged values. Returns
// ErrNoDefaultValue for unnamed values.
func (self ColorName) Default() (ColorRGB, error) { panic(notImplemented) }

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
func (self ColorSpecial) Osc4() uint16 { panic(notImplemented) }

// String returns the Zig tag name (@tagName).
func (self ColorSpecial) String() string { panic(notImplemented) }

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
func (self ColorDynamic) Next() Optional[ColorDynamic] { panic(notImplemented) }

// String returns the Zig tag name (@tagName).
func (self ColorDynamic) String() string { panic(notImplemented) }

// ColorRGB is color.RGB: RGB.
//
// Zig: packed struct(u24). The packed value is not visible to other code, so
// it is an ordinary Go struct with the same field names.
type ColorRGB struct {
	r uint8
	g uint8
	b uint8
}

// Eql is color.RGB.eql.
func (self ColorRGB) Eql(other ColorRGB) bool { panic(notImplemented) }

// Contrast is color.RGB.contrast: calculates the contrast ratio between two
// colors. The contrast ration is a value between 1 and 21 where 1 is the
// lowest contrast and 21 is the highest contrast.
//
// https://www.w3.org/TR/WCAG20/#contrast-ratiodef
func (self ColorRGB) Contrast(other ColorRGB) float64 { panic(notImplemented) }

// Luminance is color.RGB.luminance: calculates luminance based on the W3C
// formula. This returns a normalized value between 0 and 1 where 0 is black
// and 1 is white.
//
// https://www.w3.org/TR/WCAG20/#relativeluminancedef
func (self ColorRGB) Luminance() float64 { panic(notImplemented) }

// colorRGBComponentLuminance is color.RGB.componentLuminance: calculates
// single-component luminance based on the W3C formula.
//
// Expects sRGB color space which at the time of writing we don't generally
// use but it's a good enough approximation until we fix that.
// https://www.w3.org/TR/WCAG20/#relativeluminancedef
func colorRGBComponentLuminance(c uint8) float64 { panic(notImplemented) }

// PerceivedLuminance is color.RGB.perceivedLuminance: calculates "perceived
// luminance" which is better for determining light vs dark.
//
// Source: https://www.w3.org/TR/AERT/#color-contrast
func (self ColorRGB) PerceivedLuminance() float64 { panic(notImplemented) }

// colorRGBFromIntensity is color.RGB.fromIntensity: parse a color from a
// floating point intensity value.
//
// The value should be between 0.0 and 1.0, inclusive.
func colorRGBFromIntensity(value []uint8) (uint8, error) { panic(notImplemented) }

// colorRGBFromHex is color.RGB.fromHex: parse a color from a string of
// hexadecimal digits.
//
// The string can contain 1, 2, 3, or 4 characters and represents the color
// value scaled in 4, 8, 12, or 16 bits, respectively.
func colorRGBFromHex(value []uint8) (uint8, error) { panic(notImplemented) }

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
func ColorRGBParse(value []uint8) (ColorRGB, error) { panic(notImplemented) }

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
func colorLABFromRgb(rgb ColorRGB) colorLAB { panic(notImplemented) }

// toRgb is color.LAB.toRgb: LAB to RGB.
func (self colorLAB) toRgb() ColorRGB { panic(notImplemented) }

// colorLABLerp is color.LAB.lerp: linearly interpolate between two LAB
// colors component-wise. `t` is the interpolation factor (0 = a, 1 = b).
func colorLABLerp(t float32, a colorLAB, b colorLAB) colorLAB { panic(notImplemented) }
