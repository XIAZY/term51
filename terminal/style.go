// WIREFRAME for src/terminal/style.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: style.zig is a namespace file, so its declarations take the
// prefix `Style`: `style.Id` is `StyleId`, `style.default_id` is
// `StyleDefaultId`, `style.Set` is `StyleSet`. The struct `style.Style`
// itself is `Style` (not `StyleStyle`), and what is nested in it follows
// from that: `Style.Color` is `StyleColor`, `Style.Fg` is `StyleFg`.
// Declarations that are private in Zig are unexported: `Style.Flags` is
// `styleFlags`, `Style.VTFormatter` is `styleVTFormatter`,
// `Style.HtmlFormatter` is `styleHtmlFormatter`, `Style.PackedStyle` is
// `stylePackedStyle`, `Style.PackedStyle.Data` is `stylePackedStyleData`.
//
// Declarations appear in the same order as in style.zig; the stand-ins for
// config.zig are at the end.
//
// Go-memory rule (docs/mapping.md, governing rule 3): the unused imports
// `Offset` and `OffsetBuf` have no counterpart. `Set` takes no buffer; see
// ref_counted_set.go.
//
// Packed structs:
//   - `Flags` is `packed struct(u16)`. It is an ordinary struct with the
//     same field names, including `_padding`, which takes part in `eql` and
//     in `hash`. No non-test Zig code converts Flags alone to its integer,
//     so there is no conversion stub for it.
//   - `PackedStyle` is `packed struct(u128)`. `hash` converts it to two
//     64-bit words with `@bitCast`; that conversion is the stub
//     `stylePackedStyle.bitCast`, the one name in this file that is not a
//     Zig declaration.
//   - `PackedStyle.Data` is a `packed union` of three 24-bit views with no
//     tag. Go has no untagged union; it is the 24-bit integer itself.
//
// Not ported:
//   - the `comptime` block at the end of `Style` (size and representation
//     assertions on PackedStyle): compile-time layout checks.
//
// The file has no C ABI declarations.

package terminal

// StyleId is style.Id: the unique identifier for a style. This is at most
// the number of cells that can fit into a terminal page.
type StyleId = SizeStyleCountInt

// StyleDefaultId is style.default_id: the Id to use for default styling.
const StyleDefaultId StyleId = 0

// Style is style.Style: the style attributes for a cell.
//
// The Go zero value is Zig's default `.{}`: all three colors `.none` and
// every flag off.
type Style struct {
	// Various colors, all self-explanatory.
	fg_color        StyleColor
	bg_color        StyleColor
	underline_color StyleColor

	// On/off attributes that don't require much bit width so we use
	// a packed struct to make this take up significantly less space.
	flags styleFlags
}

// styleFlags is style.Style.Flags.
//
// Zig: packed struct(u16); the fields occupy the bits from the least
// significant in the order listed: bold is bit 0, overline is bit 7,
// underline is bits 8..10, _padding is bits 11..15.
type styleFlags struct {
	bold          bool
	italic        bool
	faint         bool
	blink         bool
	inverse       bool
	invisible     bool
	strikethrough bool
	overline      bool
	underline     SGRAttributeUnderline

	// Zig: u5, default 0.
	_padding uint8
}

// StyleColor is style.Style.Color: the color for an SGR attribute. A color
// can come from multiple sources so we use this to track the source plus
// color value so that we can properly react to things like palette changes.
//
// Zig: union(Tag). Only the field named by Tag is meaningful. The variant
// `none` has no payload and no field. The Go zero value is `.none`.
type StyleColor struct {
	Tag StyleColorTag

	Palette uint8
	Rgb     ColorRGB
}

// StyleColorTag is style.Style.Color.Tag.
//
// Zig: enum(u8), values numbered from zero in the order listed. `Tag` is
// private in Zig; it is exported here because it is the type of the
// exported `Tag` field.
type StyleColorTag uint8

const (
	StyleColorNone StyleColorTag = iota
	StyleColorPalette
	StyleColorRgb
)

// String returns the Zig tag name (@tagName).
func (self StyleColorTag) String() string { panic(notImplemented) }

// Eql is style.Style.Color.eql: true if the color is equal to another
// color.
func (self StyleColor) Eql(other StyleColor) bool { panic(notImplemented) }

// Format is style.Style.Color.format: formatting to make debug logs easier
// to read by only including non-default attributes.
//
// Per docs/mapping.md a Zig `format` method becomes `Format() string` with
// output identical to Zig's.
func (self StyleColor) Format() string { panic(notImplemented) }

// Default is style.Style.default: true if the style is the default style.
func (self Style) Default() bool { panic(notImplemented) }

// Eql is style.Style.eql: true if the style is equal to another style.
func (self Style) Eql(other Style) bool { panic(notImplemented) }

// Bg is style.Style.bg: returns the bg color for a cell with this style
// given the cell that has this style and the palette to use.
//
// Note that generally if a cell is a color-only cell, it SHOULD
// only have the default style, but this is meant to work with the
// default style as well.
//
// Zig: cell is *const page.Cell, palette is *const color.Palette; returns
// ?color.RGB.
func (self Style) Bg(cell *PageCell, palette *ColorPalette) Optional[ColorRGB] {
	panic(notImplemented)
}

// StyleFg is style.Style.Fg: the options of Style.fg.
type StyleFg struct {
	// The default color to use if the style doesn't specify a
	// foreground color and no configuration options override
	// it.
	//
	// Zig: `default`, a Go keyword.
	default_ ColorRGB

	// The current color palette. Required to map palette indices to
	// real color values.
	palette *ColorPalette

	// If specified, the color to use for bold text.
	//
	// Zig: ?configpkg.BoldColor, default null.
	bold Optional[ConfigBoldColor]
}

// Fg is style.Style.fg: returns the fg color for a cell with this style
// given the palette and various configuration options.
func (self Style) Fg(opts StyleFg) ColorRGB { panic(notImplemented) }

// UnderlineColor is style.Style.underlineColor: returns the underline color
// for this style.
func (self Style) UnderlineColor(palette *ColorPalette) Optional[ColorRGB] {
	panic(notImplemented)
}

// BgCell is style.Style.bgCell: returns a bg-color only cell from this
// style, if it exists.
//
// Zig: returns ?page.Cell.
func (self Style) BgCell() Optional[PageCell] { panic(notImplemented) }

// Format is style.Style.format: formatting to make debug logs easier to
// read by only including non-default attributes.
func (self Style) Format() string { panic(notImplemented) }

// FormatterVt is style.Style.formatterVt: returns a formatter that renders
// this style as VT sequences, to be used with `{f}`. This always resets the
// style first `\x1b[0m` since a style is meant to be fully self-contained.
//
// For individual styles, this always emits multiple SGR sequences
// (i.e. an individual `\x1b[<stuff>m` for each attribute) rather than
// trying to combine them into a single sequence. We do this because
// terminals have varying levels of support for combined sequences
// especially with mixed separators (e.g. `:` vs `;`).
//
// Zig: self is *const Style, and the formatter keeps that pointer.
func (self *Style) FormatterVt() styleVTFormatter { panic(notImplemented) }

// FormatterHtml is style.Style.formatterHtml: returns a formatter that
// renders this style as inline CSS properties, to be used with `{f}`. The
// output is a valid CSS style string suitable for use in a `style`
// attribute (e.g., "color: rgb(255, 0, 0); font-weight: bold;").
//
// Palette colors are emitted as CSS variables like `var(--vt-palette-N)`.
//
// Zig: self is *const Style, and the formatter keeps that pointer.
func (self *Style) FormatterHtml() styleHtmlFormatter { panic(notImplemented) }

// styleVTFormatter is style.Style.VTFormatter.
type styleVTFormatter struct {
	style *Style

	// If set, palette colors will be emitted as RGB values instead of
	// palette indices. This is useful when you want to capture the
	// exact colors at formatting time rather than relying on the
	// terminal's palette.
	//
	// Zig: ?*const color.Palette, default null.
	palette *ColorPalette
}

// Format is style.Style.VTFormatter.format.
//
// Zig: pub fn format(self: VTFormatter, writer: *std.Io.Writer) !void, used
// with `{f}`.
func (self styleVTFormatter) Format() string { panic(notImplemented) }

// formatColor is style.Style.VTFormatter.formatColor.
//
// Zig: fn formatColor(self: VTFormatter, writer: *std.Io.Writer,
// prefix: u8, value: Color) !void. The writer is oscWriter (osc.go), the
// package's stand-in for std.Io.Writer.
func (self styleVTFormatter) formatColor(writer *oscWriter, prefix uint8, value StyleColor) error {
	panic(notImplemented)
}

// styleHtmlFormatter is style.Style.HtmlFormatter.
type styleHtmlFormatter struct {
	style *Style

	// If set, palette colors will be emitted as RGB values instead of
	// CSS variables. This is useful when you want to capture the exact
	// colors at formatting time rather than relying on CSS variables.
	//
	// Zig: ?*const color.Palette, default null.
	palette *ColorPalette
}

// Format is style.Style.HtmlFormatter.format.
//
// Zig: pub fn format(self: HtmlFormatter, writer: *std.Io.Writer) !void,
// used with `{f}`.
func (self styleHtmlFormatter) Format() string { panic(notImplemented) }

// formatColor is style.Style.HtmlFormatter.formatColor.
//
// Zig: fn formatColor(self: HtmlFormatter, writer: *std.Io.Writer,
// property: []const u8, c: Color) !void. The writer is oscWriter (osc.go),
// the package's stand-in for std.Io.Writer.
func (self styleHtmlFormatter) formatColor(writer *oscWriter, property []uint8, c StyleColor) error {
	panic(notImplemented)
}

// stylePackedStyle is style.Style.PackedStyle.
//
// `PackedStyle` represents the same data as `Style` but without padding,
// which is necessary for hashing via re-interpretation of the underlying
// bytes.
//
// `Style` is still preferred for everything else as it has type-safety
// when using the `Color` tagged union.
//
// Empirical testing shows that storing all of the tags first and then the
// data provides a better layout for serializing into and is faster on
// benchmarks.
//
// Zig: packed struct(u128); the fields occupy the bits from the least
// significant in the order listed: tags.fg bits 0..7, tags.bg 8..15,
// tags.underline 16..23, data.fg 24..47, data.bg 48..71, data.underline
// 72..95, flags 96..111, _padding 112..127.
type stylePackedStyle struct {
	tags struct {
		fg        StyleColorTag
		bg        StyleColorTag
		underline StyleColorTag
	}
	data struct {
		fg        stylePackedStyleData
		bg        stylePackedStyleData
		underline stylePackedStyleData
	}
	flags styleFlags

	// Zig: u16, default 0.
	_padding uint16
}

// stylePackedStyleData is style.Style.PackedStyle.Data.
//
// After https://github.com/ziglang/zig/issues/19754 is implemented,
// it will be an compiler-error to have packed union fields of
// differing size.
//
// For now we just need to be careful not to accidentally introduce
// padding.
//
// Zig: a packed union of three 24-bit views: `none: u24`,
// `palette: packed struct(u24) { idx: u8, _padding: u16 = 0 }` and
// `rgb: color.RGB`. Go has no untagged union, so this is the 24-bit integer
// itself (u24, held in the next larger Go type): `none` is 0, `palette` has
// idx in bits 0..7, `rgb` has r in bits 0..7, g in bits 8..15 and b in bits
// 16..23.
type stylePackedStyleData uint32

// stylePackedStyleDataFromColor is style.Style.PackedStyle.Data.fromColor.
func stylePackedStyleDataFromColor(c StyleColor) stylePackedStyleData { panic(notImplemented) }

// stylePackedStyleFromStyle is style.Style.PackedStyle.fromStyle.
func stylePackedStyleFromStyle(style Style) stylePackedStyle { panic(notImplemented) }

// bitCast is not a Zig declaration. It stands for
// `const wide: [2]u64 = @bitCast(packed_style);` in Style.hash (style.zig
// line 530): the 128 bits of the packed struct, element 0 holding bits
// 0..63 and element 1 holding bits 64..127.
func (self stylePackedStyle) bitCast() [2]uint64 { panic(notImplemented) }

// Hash is style.Style.hash.
//
// Zig: self is *const Style.
func (self *Style) Hash() uint64 { panic(notImplemented) }

// StyleSet is style.Set: the RefCountedSet instantiation that stores
// styles.
//
// Zig: RefCountedSet(Style, Id, size.CellCountInt, struct { hash, eql }).
type StyleSet = RefCountedSet[Style, StyleId, SizeCellCountInt, StyleSetContext]

// StyleSetContext is the anonymous context struct of style.Set, reachable
// in Zig as `Set.Context`. It has no fields and no `deleted` function.
type StyleSetContext struct{}

// Hash is the context's `hash`: Style.hash of the style.
//
// Zig: pub fn hash(self: *const @This(), style: Style) u64. See
// RefCountedSetContext for the receiver.
func (self StyleSetContext) Hash(style Style) uint64 { panic(notImplemented) }

// Eql is the context's `eql`: Style.eql of the two styles.
//
// Zig: pub fn eql(self: *const @This(), a: Style, b: Style) bool.
func (self StyleSetContext) Eql(a Style, b Style) bool { panic(notImplemented) }

// Stand-ins for config.zig, which is outside the library and has no Go
// file. Types and constants only, as far as the signature of Style.fg needs
// them.

// ConfigBoldColor is config.BoldColor (src/config.zig line 19, which
// re-exports Config.BoldColor, src/config/Config.zig line 5575): represents
// color values that can be used for bold. See `bold-color`.
//
// Zig: union(enum) { color: Color, bright }. Only the field named by Tag is
// meaningful; `bright` has no payload and no field. Its functions
// `parseCLI` and `formatEntry` are configuration-file code and are not
// declared.
type ConfigBoldColor struct {
	Tag ConfigBoldColorTag

	Color ConfigColor
}

// ConfigBoldColorTag is the tag of config.BoldColor, values numbered from
// zero in the order listed.
type ConfigBoldColorTag uint8

const (
	ConfigBoldColorColor ConfigBoldColorTag = iota
	ConfigBoldColorBright
)

// ConfigColor is config.Config.Color (src/config/Config.zig line 5374), the
// payload of BoldColor.color.
//
// Style.fg calls its `toTerminalRGB` (Config.zig line 5391), which copies
// r, g and b into a color.RGB. That function, like the rest of the type's
// functions, is not declared here.
type ConfigColor struct {
	r uint8
	g uint8
	b uint8
}

// ToTerminalRGB is config.Color.toTerminalRGB (Config.zig): the same
// channels as a terminal color. Style.fg calls it for a bold color.
func (self ConfigColor) ToTerminalRGB() ColorRGB { panic(notImplemented) }
