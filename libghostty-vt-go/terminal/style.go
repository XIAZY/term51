// Ported from src/terminal/style.zig (Ghostty v1.3.1).
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
type StyleID = SizeStyleCountInt

// StyleDefaultId is style.default_id: the Id to use for default styling.
const StyleDefaultID StyleID = 0

// Style is style.Style: the style attributes for a cell.
//
// The Go zero value is Zig's default `.{}`: all three colors `.none` and
// every flag off.
type Style struct {
	// Various colors, all self-explanatory.
	FgColor         StyleColor
	BgColor         StyleColor
	underline_color StyleColor

	// On/off attributes that don't require much bit width so we use
	// a packed struct to make this take up significantly less space.
	Flags StyleFlags
}

// styleFlags is style.Style.Flags.
//
// Zig: packed struct(u16); the fields occupy the bits from the least
// significant in the order listed: bold is bit 0, overline is bit 7,
// underline is bits 8..10, _padding is bits 11..15.
type StyleFlags struct {
	Bold          bool
	Italic        bool
	Faint         bool
	Blink         bool
	Inverse       bool
	Invisible     bool
	Strikethrough bool
	Overline      bool
	Underline     SGRAttributeUnderline

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

// styleColorTagNames are the tag names of StyleColorTag, indexed by value
// (Zig's @tagName).
var styleColorTagNames = [...]string{"none", "palette", "rgb"}

// String returns the Zig tag name (@tagName).
func (self StyleColorTag) String() string {
	return styleColorTagNames[self]
}

// Eql is style.Style.Color.eql: true if the color is equal to another
// color.
func (self StyleColor) Eql(other StyleColor) bool {
	// Zig: `@as(Tag, self) == @as(Tag, other) and switch (self) {...}`.
	if self.Tag != other.Tag {
		return false
	}
	switch self.Tag {
	case StyleColorNone:
		return true
	case StyleColorPalette:
		return self.Palette == other.Palette
	case StyleColorRgb:
		return self.Rgb == other.Rgb
	}
	panic("unreachable")
}

// Format is style.Style.Color.format: formatting to make debug logs easier
// to read by only including non-default attributes.
//
// Per docs/mapping.md a Zig `format` method becomes `Format() string` with
// output identical to Zig's.
func (self StyleColor) Format() string {
	// Zig: `_ = fmt; _ = options;` the format string and options are
	// unused, and Go's Format has none.
	switch self.Tag {
	case StyleColorNone:
		return "Color.none"
	case StyleColorPalette:
		p := self.Palette
		return "Color.palette{ " + itoa(uint64(p)) + " }"
	case StyleColorRgb:
		rgb := self.Rgb
		return "Color.rgb{ " + itoa(uint64(rgb.R)) + ", " + itoa(uint64(rgb.G)) + ", " + itoa(uint64(rgb.B)) + " }"
	}
	panic("unreachable")
}

// Default is style.Style.default: true if the style is the default style.
func (self Style) Default() bool {
	return self.Eql(Style{})
}

// Eql is style.Style.eql: true if the style is equal to another style.
//
// Zig: `self.flags == other.flags` compares the packed u16. Go's struct ==
// would also compare the bits above each field's Zig width (underline is a
// u3 and _padding a u5 in Zig, whole bytes in Go), so the flags are compared
// as the same masked u16 that Style.hash uses (styleFlags.bits).
func (self Style) Eql(other Style) bool {
	return self.Flags.bits() == other.Flags.bits() &&
		self.FgColor.Eql(other.FgColor) &&
		self.BgColor.Eql(other.BgColor) &&
		self.underline_color.Eql(other.underline_color)
}

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
	switch cell.ContentTag() {
	case PageCellContentTagBgColorPalette:
		return Some(palette[cell.content().colorPalette()])
	case PageCellContentTagBgColorRgb:
		rgb := cell.content().colorRGB()
		return Some(ColorRGB{R: rgb.R, G: rgb.G, B: rgb.B})

	default:
		switch self.BgColor.Tag {
		case StyleColorNone:
			return None[ColorRGB]()
		case StyleColorPalette:
			idx := self.BgColor.Palette
			return Some(palette[idx])
		case StyleColorRgb:
			return Some(self.BgColor.Rgb)
		}
		panic("unreachable")
	}
}

// StyleFg is style.Style.Fg: the options of Style.fg.
type StyleFg struct {
	// The default color to use if the style doesn't specify a
	// foreground color and no configuration options override
	// it.
	//
	// Zig: `default`, a Go keyword.
	Default ColorRGB

	// The current color palette. Required to map palette indices to
	// real color values.
	Palette *ColorPalette

	// If specified, the color to use for bold text.
	//
	// Zig: ?configpkg.BoldColor, default null.
	Bold Optional[ConfigBoldColor]
}

// Fg is style.Style.fg: returns the fg color for a cell with this style
// given the palette and various configuration options.
func (self Style) Fg(opts StyleFg) ColorRGB {
	// Note we don't pull the bold check to the top-level here because
	// we don't want to duplicate the conditional multiple times since
	// certain colors require more checks (e.g. `bold_is_bright`).

	switch self.FgColor.Tag {
	case StyleColorNone: // default:
		if self.Flags.Bold {
			if opts.Bold.Valid {
				bold := opts.Bold.Value
				switch bold.Tag {
				case ConfigBoldColorBright:
				case ConfigBoldColorColor:
					v := bold.Color
					return v.ToTerminalRGB() // break :default
				default:
					panic("unreachable")
				}
			}
		}

		return opts.Default // break :default

	case StyleColorPalette: // palette:
		idx := self.FgColor.Palette
		if self.Flags.Bold {
			if opts.Bold.Valid {
				const bright_offset = uint8(ColorNameBrightBlack) // @intFromEnum(color.Name.bright_black)
				if idx < bright_offset {
					return opts.Palette[idx+bright_offset] // break :palette
				}
			}
		}

		return opts.Palette[idx] // break :palette

	case StyleColorRgb: // rgb:
		rgb := self.FgColor.Rgb
		if self.Flags.Bold && rgb.Eql(opts.Default) {
			if opts.Bold.Valid {
				bold := opts.Bold.Value
				switch bold.Tag {
				case ConfigBoldColorColor:
					v := bold.Color
					return v.ToTerminalRGB() // break :rgb
				case ConfigBoldColorBright:
				default:
					panic("unreachable")
				}
			}
		}

		return rgb // break :rgb
	}
	panic("unreachable")
}

// UnderlineColor is style.Style.underlineColor: returns the underline color
// for this style.
func (self Style) UnderlineColor(palette *ColorPalette) Optional[ColorRGB] {
	switch self.underline_color.Tag {
	case StyleColorNone:
		return None[ColorRGB]()
	case StyleColorPalette:
		idx := self.underline_color.Palette
		return Some(palette[idx])
	case StyleColorRgb:
		return Some(self.underline_color.Rgb)
	}
	panic("unreachable")
}

// BgCell is style.Style.bgCell: returns a bg-color only cell from this
// style, if it exists.
//
// Zig: returns ?page.Cell.
func (self Style) BgCell() Optional[PageCell] {
	switch self.BgColor.Tag {
	case StyleColorNone:
		return None[PageCell]()
	case StyleColorPalette:
		idx := self.BgColor.Palette
		return Some(pageCellFields{
			content_tag: PageCellContentTagBgColorPalette,
			content:     pageCellContentFields{color_palette: idx}.pack(),
		}.pack())
	case StyleColorRgb:
		rgb := self.BgColor.Rgb
		return Some(pageCellFields{
			content_tag: PageCellContentTagBgColorRgb,
			content: pageCellContentFields{color_rgb: PageCellRGB{
				R: rgb.R,
				G: rgb.G,
				B: rgb.B,
			}}.pack(),
		}.pack())
	}
	panic("unreachable")
}

// Format is style.Style.format: formatting to make debug logs easier to
// read by only including non-default attributes.
func (self Style) Format() string {
	// Zig: `_ = fmt; _ = options;` the format string and options are
	// unused, and Go's Format has none. The writer is the returned string.
	var writer []uint8

	dflt := Style{}

	// Go-only: print the flags as their Zig bits (styleFlags.bits), so a Go
	// value with bits above underline's u3 or _padding's u5 prints as it
	// compares and hashes. self is a copy.
	self.Flags.Underline, self.Flags._padding = SGRAttributeUnderline(self.Flags.bits()>>8&0x7), uint8(self.Flags.bits()>>11)

	writer = append(writer, "Style{ "...)

	started := false

	// Zig: `inline for (std.meta.fields(Style))`: fg_color, bg_color,
	// underline_color, then flags, which is handled in place. Every field
	// that is not `flags` is a Color: `!std.meta.eql(value, d_val)` is
	// Color.eql (tag, then the active payload), and `{any}` of it is
	// fmtAny below.
	//
	// fmtAny is `{any}` of a Color (std.Io.Writer.printValue, Zig 0.15.2,
	// max_depth 3): a tagged union prints `.{ .<tag> = <payload> }`, a void
	// payload `void`, a u8 in decimal and color.RGB, a packed struct, as
	// `.{ .r = R, .g = G, .b = B }`.
	fmtAny := func(c StyleColor) string {
		var payload string
		switch c.Tag {
		case StyleColorNone:
			payload = "void"
		case StyleColorPalette:
			payload = itoa(uint64(c.Palette))
		case StyleColorRgb:
			payload = ".{ .r = " + itoa(uint64(c.Rgb.R)) + ", .g = " + itoa(uint64(c.Rgb.G)) +
				", .b = " + itoa(uint64(c.Rgb.B)) + " }"
		default:
			panic("unreachable")
		}
		return ".{ ." + c.Tag.String() + " = " + payload + " }"
	}
	for _, f := range [...]struct {
		name  string
		value StyleColor
		d_val StyleColor
	}{
		{"fg_color", self.FgColor, dflt.FgColor},
		{"bg_color", self.BgColor, dflt.BgColor},
		{"underline_color", self.underline_color, dflt.underline_color},
	} {
		value := f.value
		d_val := f.d_val
		if !value.Eql(d_val) {
			if started {
				writer = append(writer, ", "...)
			}
			writer = append(writer, f.name+"="+fmtAny(value)...)
			started = true
		}
	}

	// The field `flags`, last in Style.
	{
		if started {
			writer = append(writer, ", "...)
		}

		writer = append(writer, "flags={ "...)

		started = false

		// Zig: `inline for (std.meta.fields(@TypeOf(self.flags)))`: the
		// eight bools in order, then underline and _padding, which are not
		// bool and are printed with `{any}` when they differ from the
		// default.
		for _, ff := range [...]struct {
			name string
			v    bool
		}{
			{"bold", self.Flags.Bold},
			{"italic", self.Flags.Italic},
			{"faint", self.Flags.Faint},
			{"blink", self.Flags.Blink},
			{"inverse", self.Flags.Inverse},
			{"invisible", self.Flags.Invisible},
			{"strikethrough", self.Flags.Strikethrough},
			{"overline", self.Flags.Overline},
		} {
			if ff.v {
				if started {
					writer = append(writer, ", "...)
				}
				writer = append(writer, ff.name...)
				started = true
			}
		}
		if self.Flags.Underline != dflt.Flags.Underline {
			if started {
				writer = append(writer, ", "...)
			}
			// `{any}` of an exhaustive enum: `.` and the tag name.
			writer = append(writer, "underline=."+self.Flags.Underline.String()...)
			started = true
		}
		if self.Flags._padding != dflt.Flags._padding {
			if started {
				writer = append(writer, ", "...)
			}
			// `{any}` of a u5: decimal.
			writer = append(writer, "_padding="+itoa(uint64(self.Flags._padding))...)
			started = true
		}
		writer = append(writer, " }"...)

		started = true
	}

	writer = append(writer, " }"...)
	return string(writer)
}

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
func (self *Style) FormatterVt() styleVTFormatter {
	return styleVTFormatter{style: self}
}

// FormatterHtml is style.Style.formatterHtml: returns a formatter that
// renders this style as inline CSS properties, to be used with `{f}`. The
// output is a valid CSS style string suitable for use in a `style`
// attribute (e.g., "color: rgb(255, 0, 0); font-weight: bold;").
//
// Palette colors are emitted as CSS variables like `var(--vt-palette-N)`.
//
// Zig: self is *const Style, and the formatter keeps that pointer.
func (self *Style) FormatterHtml() styleHtmlFormatter {
	return styleHtmlFormatter{style: self}
}

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
func (self styleVTFormatter) Format() string {
	builder := oscWriterAllocatingInit()
	format := func(writer *oscWriter) error {
		// Always reset the style. Styles are fully self-contained.
		// Even if this style is empty, then that means we want to go
		// back to the default.
		if err := writer.writeAll([]uint8("\x1b[0m")); err != nil {
			return err
		}

		// Our flags
		if self.style.Flags.Bold {
			if err := writer.writeAll([]uint8("\x1b[1m")); err != nil {
				return err
			}
		}
		if self.style.Flags.Faint {
			if err := writer.writeAll([]uint8("\x1b[2m")); err != nil {
				return err
			}
		}
		if self.style.Flags.Italic {
			if err := writer.writeAll([]uint8("\x1b[3m")); err != nil {
				return err
			}
		}
		if self.style.Flags.Blink {
			if err := writer.writeAll([]uint8("\x1b[5m")); err != nil {
				return err
			}
		}
		if self.style.Flags.Inverse {
			if err := writer.writeAll([]uint8("\x1b[7m")); err != nil {
				return err
			}
		}
		if self.style.Flags.Invisible {
			if err := writer.writeAll([]uint8("\x1b[8m")); err != nil {
				return err
			}
		}
		if self.style.Flags.Strikethrough {
			if err := writer.writeAll([]uint8("\x1b[9m")); err != nil {
				return err
			}
		}
		if self.style.Flags.Overline {
			if err := writer.writeAll([]uint8("\x1b[53m")); err != nil {
				return err
			}
		}
		switch self.style.Flags.Underline {
		case SGRAttributeUnderlineNone:
		case SGRAttributeUnderlineSingle:
			if err := writer.writeAll([]uint8("\x1b[4m")); err != nil {
				return err
			}
		case SGRAttributeUnderlineDouble:
			if err := writer.writeAll([]uint8("\x1b[4:2m")); err != nil {
				return err
			}
		case SGRAttributeUnderlineCurly:
			if err := writer.writeAll([]uint8("\x1b[4:3m")); err != nil {
				return err
			}
		case SGRAttributeUnderlineDotted:
			if err := writer.writeAll([]uint8("\x1b[4:4m")); err != nil {
				return err
			}
		case SGRAttributeUnderlineDashed:
			if err := writer.writeAll([]uint8("\x1b[4:5m")); err != nil {
				return err
			}
		default:
			panic("unreachable")
		}

		// Various RGB colors.
		if err := self.formatColor(writer, 38, self.style.FgColor); err != nil {
			return err
		}
		if err := self.formatColor(writer, 48, self.style.BgColor); err != nil {
			return err
		}
		if err := self.formatColor(writer, 58, self.style.underline_color); err != nil {
			return err
		}
		return nil
	}
	// An allocating writer never fails.
	if err := format(&builder.writer); err != nil {
		panic("unreachable")
	}
	return string(builder.written())
}

// formatColor is style.Style.VTFormatter.formatColor.
//
// Zig: fn formatColor(self: VTFormatter, writer: *std.Io.Writer,
// prefix: u8, value: Color) !void. The writer is oscWriter (osc.go), the
// package's stand-in for std.Io.Writer.
func (self styleVTFormatter) formatColor(writer *oscWriter, prefix uint8, value StyleColor) error {
	switch value.Tag {
	case StyleColorNone:
	case StyleColorPalette:
		idx := value.Palette
		if self.palette != nil {
			p := self.palette
			rgb := p[idx]
			// writer.print("\x1b[{d};2;{d};{d};{d}m", .{ prefix, rgb.r, rgb.g, rgb.b })
			return writer.writeAll([]uint8("\x1b[" + itoa(uint64(prefix)) + ";2;" +
				itoa(uint64(rgb.R)) + ";" + itoa(uint64(rgb.G)) + ";" + itoa(uint64(rgb.B)) + "m"))
		} else {
			// writer.print("\x1b[{d};5;{d}m", .{ prefix, idx })
			return writer.writeAll([]uint8("\x1b[" + itoa(uint64(prefix)) + ";5;" + itoa(uint64(idx)) + "m"))
		}
	case StyleColorRgb:
		rgb := value.Rgb
		// writer.print("\x1b[{d};2;{d};{d};{d}m", .{ prefix, rgb.r, rgb.g, rgb.b })
		return writer.writeAll([]uint8("\x1b[" + itoa(uint64(prefix)) + ";2;" +
			itoa(uint64(rgb.R)) + ";" + itoa(uint64(rgb.G)) + ";" + itoa(uint64(rgb.B)) + "m"))
	default:
		panic("unreachable")
	}
	return nil
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
func (self styleHtmlFormatter) Format() string {
	builder := oscWriterAllocatingInit()
	format := func(writer *oscWriter) error {
		// Colors
		if err := self.formatColor(writer, []uint8("color"), self.style.FgColor); err != nil {
			return err
		}
		if err := self.formatColor(writer, []uint8("background-color"), self.style.BgColor); err != nil {
			return err
		}
		if err := self.formatColor(writer, []uint8("text-decoration-color"), self.style.underline_color); err != nil {
			return err
		}

		// Text decoration line
		has_line := self.style.Flags.Underline != SGRAttributeUnderlineNone ||
			self.style.Flags.Strikethrough ||
			self.style.Flags.Overline ||
			self.style.Flags.Blink
		if has_line {
			if err := writer.writeAll([]uint8("text-decoration-line:")); err != nil {
				return err
			}
			if self.style.Flags.Underline != SGRAttributeUnderlineNone {
				if err := writer.writeAll([]uint8(" underline")); err != nil {
					return err
				}
			}
			if self.style.Flags.Strikethrough {
				if err := writer.writeAll([]uint8(" line-through")); err != nil {
					return err
				}
			}
			if self.style.Flags.Overline {
				if err := writer.writeAll([]uint8(" overline")); err != nil {
					return err
				}
			}
			if self.style.Flags.Blink {
				if err := writer.writeAll([]uint8(" blink")); err != nil {
					return err
				}
			}
			if err := writer.writeAll([]uint8(";")); err != nil {
				return err
			}
		}

		// Text decoration style
		switch self.style.Flags.Underline {
		case SGRAttributeUnderlineNone:
		case SGRAttributeUnderlineSingle:
			if err := writer.writeAll([]uint8("text-decoration-style: solid;")); err != nil {
				return err
			}
		case SGRAttributeUnderlineDouble:
			if err := writer.writeAll([]uint8("text-decoration-style: double;")); err != nil {
				return err
			}
		case SGRAttributeUnderlineCurly:
			if err := writer.writeAll([]uint8("text-decoration-style: wavy;")); err != nil {
				return err
			}
		case SGRAttributeUnderlineDotted:
			if err := writer.writeAll([]uint8("text-decoration-style: dotted;")); err != nil {
				return err
			}
		case SGRAttributeUnderlineDashed:
			if err := writer.writeAll([]uint8("text-decoration-style: dashed;")); err != nil {
				return err
			}
		default:
			panic("unreachable")
		}

		if self.style.Flags.Bold {
			if err := writer.writeAll([]uint8("font-weight: bold;")); err != nil {
				return err
			}
		}
		if self.style.Flags.Italic {
			if err := writer.writeAll([]uint8("font-style: italic;")); err != nil {
				return err
			}
		}
		if self.style.Flags.Faint {
			if err := writer.writeAll([]uint8("opacity: 0.5;")); err != nil {
				return err
			}
		}
		if self.style.Flags.Invisible {
			if err := writer.writeAll([]uint8("visibility: hidden;")); err != nil {
				return err
			}
		}
		if self.style.Flags.Inverse {
			if err := writer.writeAll([]uint8("filter: invert(100%);")); err != nil {
				return err
			}
		}
		return nil
	}
	// An allocating writer never fails.
	if err := format(&builder.writer); err != nil {
		panic("unreachable")
	}
	return string(builder.written())
}

// formatColor is style.Style.HtmlFormatter.formatColor.
//
// Zig: fn formatColor(self: HtmlFormatter, writer: *std.Io.Writer,
// property: []const u8, c: Color) !void. The writer is oscWriter (osc.go),
// the package's stand-in for std.Io.Writer.
func (self styleHtmlFormatter) formatColor(writer *oscWriter, property []uint8, c StyleColor) error {
	switch c.Tag {
	case StyleColorNone:
	case StyleColorPalette:
		idx := c.Palette
		if self.palette != nil {
			p := self.palette
			rgb := p[idx]
			// writer.print("{s}: rgb({d}, {d}, {d});", .{ property, rgb.r, rgb.g, rgb.b })
			return writer.writeAll([]uint8(string(property) + ": rgb(" +
				itoa(uint64(rgb.R)) + ", " + itoa(uint64(rgb.G)) + ", " + itoa(uint64(rgb.B)) + ");"))
		} else {
			// writer.print("{s}: var(--vt-palette-{d});", .{ property, idx })
			return writer.writeAll([]uint8(string(property) + ": var(--vt-palette-" + itoa(uint64(idx)) + ");"))
		}
	case StyleColorRgb:
		rgb := c.Rgb
		// writer.print("{s}: rgb({d}, {d}, {d});", .{ property, rgb.r, rgb.g, rgb.b })
		return writer.writeAll([]uint8(string(property) + ": rgb(" +
			itoa(uint64(rgb.R)) + ", " + itoa(uint64(rgb.G)) + ", " + itoa(uint64(rgb.B)) + ");"))
	default:
		panic("unreachable")
	}
	return nil
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
	flags StyleFlags

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
func stylePackedStyleDataFromColor(c StyleColor) stylePackedStyleData {
	// Zig: `inline else => |v, t| @unionInit(Data, @tagName(t), ...)`:
	// the view of Data named like the color's tag, as its 24 bits.
	switch c.Tag {
	case StyleColorNone:
		return 0 // .none = 0
	case StyleColorPalette:
		v := c.Palette
		return stylePackedStyleData(v) // .palette = .{ .idx = v } (_padding 0)
	case StyleColorRgb:
		v := c.Rgb
		return stylePackedStyleData(v.R) | stylePackedStyleData(v.G)<<8 | stylePackedStyleData(v.B)<<16 // .rgb = v
	}
	panic("unreachable")
}

// stylePackedStyleFromStyle is style.Style.PackedStyle.fromStyle.
func stylePackedStyleFromStyle(style Style) stylePackedStyle {
	var self stylePackedStyle
	self.tags.fg = style.FgColor.Tag
	self.tags.bg = style.BgColor.Tag
	self.tags.underline = style.underline_color.Tag
	self.data.fg = stylePackedStyleDataFromColor(style.FgColor)
	self.data.bg = stylePackedStyleDataFromColor(style.BgColor)
	self.data.underline = stylePackedStyleDataFromColor(style.underline_color)
	self.flags = style.Flags
	// _padding: Zig default 0.
	return self
}

// bits is not a Zig declaration. It is `@as(u16, @bitCast(flags))`:
// Style.Flags as its u16, bold bit 0 .. overline bit 7, underline bits 8..10
// (u3), _padding bits 11..15 (u5). The masks keep each field to its Zig
// width, so a Go value with bits above a field's width (which Zig cannot
// hold) compares and hashes as the Zig value of those bits.
func (f StyleFlags) bits() uint16 {
	b := func(v bool) uint16 {
		if v {
			return 1
		}
		return 0
	}
	return b(f.Bold)<<0 | b(f.Italic)<<1 | b(f.Faint)<<2 | b(f.Blink)<<3 |
		b(f.Inverse)<<4 | b(f.Invisible)<<5 | b(f.Strikethrough)<<6 | b(f.Overline)<<7 |
		(uint16(f.Underline)&0x7)<<8 | (uint16(f._padding)&0x1f)<<11
}

// bitCast is not a Zig declaration. It stands for
// `const wide: [2]u64 = @bitCast(packed_style);` in Style.hash (style.zig
// line 530): the 128 bits of the packed struct, element 0 holding bits
// 0..63 and element 1 holding bits 64..127.
func (self stylePackedStyle) bitCast() [2]uint64 {
	// Style.Flags as its u16, each field masked to its Zig width.
	flags := uint64(self.flags.bits())

	// The 128 bits: tags at 0, 8, 16; data (u24 each, masked) at 24, 48,
	// 72; flags at 96; _padding at 112.
	fg := uint64(self.data.fg) & 0xFFFFFF
	bg := uint64(self.data.bg) & 0xFFFFFF
	ul := uint64(self.data.underline) & 0xFFFFFF
	lo := uint64(self.tags.fg) | uint64(self.tags.bg)<<8 | uint64(self.tags.underline)<<16 |
		fg<<24 | bg<<48
	hi := bg>>16 | ul<<8 | flags<<32 | uint64(self._padding)<<48
	return [2]uint64{lo, hi}
}

// Hash is style.Style.hash.
//
// Zig: self is *const Style.
func (self *Style) Hash() uint64 {
	// We pack the style in to 128 bits, fold it to 64 bits,
	// then use std.hash.int to make it sufficiently uniform.
	packed_style := stylePackedStyleFromStyle(*self)
	wide := packed_style.bitCast()
	return stdHashIntU64(wide[0] ^ wide[1]) // @call(.always_inline, std.hash.int, ...)
}

// StyleSet is style.Set: the RefCountedSet instantiation that stores
// styles.
//
// Zig: RefCountedSet(Style, Id, size.CellCountInt, struct { hash, eql }).
type StyleSet = RefCountedSet[Style, StyleID, SizeCellCountInt, StyleSetContext]

// StyleSetContext is the anonymous context struct of style.Set, reachable
// in Zig as `Set.Context`. It has no fields and no `deleted` function.
type StyleSetContext struct{}

// Hash is the context's `hash`: Style.hash of the style.
//
// Zig: pub fn hash(self: *const @This(), style: Style) u64. See
// RefCountedSetContext for the receiver.
func (self StyleSetContext) Hash(style Style) uint64 {
	_ = self
	return style.Hash()
}

// Eql is the context's `eql`: Style.eql of the two styles.
//
// Zig: pub fn eql(self: *const @This(), a: Style, b: Style) bool.
func (self StyleSetContext) Eql(a Style, b Style) bool {
	_ = self
	return a.Eql(b)
}

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
	R uint8
	G uint8
	B uint8
}

// ToTerminalRGB is config.Color.toTerminalRGB (Config.zig): the same
// channels as a terminal color. Style.fg calls it for a bold color.
func (self ConfigColor) ToTerminalRGB() ColorRGB {
	return ColorRGB{R: self.R, G: self.G, B: self.B}
}
