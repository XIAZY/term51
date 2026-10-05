// Port of src/terminal/sgr.zig (Ghostty v1.3.1).
//
// SGR (Select Graphic Rendition) attribute parsing and types.
//
// File naming: sgr.zig is a namespace file, so its declarations take the
// prefix `SGR`.
//
// Declarations appear in the same order as in sgr.zig. The C ABI
// declarations are not ported; see docs/mapping.md. Those are:
// `lib_target`, `Attribute.Unknown.C`, `Attribute.Unknown.cval`,
// `Attribute.Underline.C`, `Attribute.Underline.cval`, `Attribute.c_union`,
// `Attribute.Value`, `Attribute.C`, `Attribute.CValue` and `Attribute.cval`.
// The file-level test helpers `testParse` and `testParseColon` are used only
// by tests and belong in sgr_test.go.

package terminal

// SGRAttribute is sgr.Attribute: attribute type for SGR.
//
// Zig: a tagged union, union(Tag). Only the field named by Tag is
// meaningful. Variants with no payload have no field.
//
// Zig's variant names `256_underline_color`, `8_bg`, `8_fg`, `8_bright_bg`,
// `8_bright_fg`, `256_bg` and `256_fg` begin with a digit, which a Go
// identifier cannot. Their payload fields take the prefix `X`. The tag
// constants are unaffected because they already begin with `SGRAttributeTag`.
type SGRAttribute struct {
	Tag SGRAttributeTag

	// Unknown attribute, the raw CSI command parameters are here.
	Unknown SGRAttributeUnknown

	// Underline the text
	Underline          SGRAttributeUnderline
	UnderlineColor     ColorRGB
	X256UnderlineColor uint8

	// Set foreground color as RGB values.
	DirectColorFg ColorRGB

	// Set background color as RGB values.
	DirectColorBg ColorRGB

	// Set the background/foreground as a named color attribute.
	X8Bg ColorName
	X8Fg ColorName

	// Set the background/foreground as a named bright color attribute.
	X8BrightBg ColorName
	X8BrightFg ColorName

	// Set background color as 256-color palette.
	X256Bg uint8

	// Set foreground color as 256-color palette.
	X256Fg uint8
}

// SGRAttributeTag is sgr.Attribute.Tag.
//
// Zig: lib.Enum(lib_target, ...), which for the Zig target is an ordinary
// enum whose values are numbered from zero in the order listed.
//
// The constants are named `SGRAttributeTag<Value>` and not
// `SGRAttribute<Value>` as for other unions, because `SGRAttributeUnknown`
// and `SGRAttributeUnderline` are the names of the payload types.
type SGRAttributeTag uint8

const (
	// Unset all attributes
	SGRAttributeTagUnset SGRAttributeTag = iota

	// Unknown attribute, the raw CSI command parameters are here.
	SGRAttributeTagUnknown

	// Bold the text.
	SGRAttributeTagBold
	SGRAttributeTagResetBold

	// Italic text.
	SGRAttributeTagItalic
	SGRAttributeTagResetItalic

	// Faint/dim text.
	// Note: reset faint is the same SGR code as reset bold
	SGRAttributeTagFaint

	// Underline the text
	SGRAttributeTagUnderline
	SGRAttributeTagUnderlineColor
	SGRAttributeTag256UnderlineColor
	SGRAttributeTagResetUnderlineColor

	// Overline the text
	SGRAttributeTagOverline
	SGRAttributeTagResetOverline

	// Blink the text
	SGRAttributeTagBlink
	SGRAttributeTagResetBlink

	// Invert fg/bg colors.
	SGRAttributeTagInverse
	SGRAttributeTagResetInverse

	// Invisible
	SGRAttributeTagInvisible
	SGRAttributeTagResetInvisible

	// Strikethrough the text.
	SGRAttributeTagStrikethrough
	SGRAttributeTagResetStrikethrough

	// Set foreground color as RGB values.
	SGRAttributeTagDirectColorFg

	// Set background color as RGB values.
	SGRAttributeTagDirectColorBg

	// Set the background/foreground as a named color attribute.
	SGRAttributeTag8Bg
	SGRAttributeTag8Fg

	// Reset the fg/bg to their default values.
	SGRAttributeTagResetFg
	SGRAttributeTagResetBg

	// Set the background/foreground as a named bright color attribute.
	SGRAttributeTag8BrightBg
	SGRAttributeTag8BrightFg

	// Set background color as 256-color palette.
	SGRAttributeTag256Bg

	// Set foreground color as 256-color palette.
	SGRAttributeTag256Fg
)

// sgrAttributeTagNames are the tag names of SGRAttributeTag, in order.
var sgrAttributeTagNames = [...]string{
	"unset",
	"unknown",
	"bold",
	"reset_bold",
	"italic",
	"reset_italic",
	"faint",
	"underline",
	"underline_color",
	"256_underline_color",
	"reset_underline_color",
	"overline",
	"reset_overline",
	"blink",
	"reset_blink",
	"inverse",
	"reset_inverse",
	"invisible",
	"reset_invisible",
	"strikethrough",
	"reset_strikethrough",
	"direct_color_fg",
	"direct_color_bg",
	"8_bg",
	"8_fg",
	"reset_fg",
	"reset_bg",
	"8_bright_bg",
	"8_bright_fg",
	"256_bg",
	"256_fg",
}

// String returns the Zig tag name (@tagName).
func (self SGRAttributeTag) String() string { return sgrAttributeTagNames[self] }

// SGRAttributeUnknown is sgr.Attribute.Unknown.
type SGRAttributeUnknown struct {
	// Full is the full SGR input.
	full []uint16

	// Partial is the remaining, where we got hung up.
	partial []uint16
}

// SGRAttributeUnderline is sgr.Attribute.Underline.
//
// Zig: enum(u3).
type SGRAttributeUnderline uint8

const (
	SGRAttributeUnderlineNone   SGRAttributeUnderline = 0
	SGRAttributeUnderlineSingle SGRAttributeUnderline = 1
	SGRAttributeUnderlineDouble SGRAttributeUnderline = 2
	SGRAttributeUnderlineCurly  SGRAttributeUnderline = 3
	SGRAttributeUnderlineDotted SGRAttributeUnderline = 4
	SGRAttributeUnderlineDashed SGRAttributeUnderline = 5
)

// sgrAttributeUnderlineNames are the tag names of SGRAttributeUnderline,
// in order.
var sgrAttributeUnderlineNames = [...]string{
	"none",
	"single",
	"double",
	"curly",
	"dotted",
	"dashed",
}

// String returns the Zig tag name (@tagName).
func (self SGRAttributeUnderline) String() string { return sgrAttributeUnderlineNames[self] }

// SGRParser is sgr.Parser: parser parses the attributes from a list of SGR
// parameters.
//
// Zig gives every field a default: `params = &.{}`,
// `params_sep = .initEmpty()` and `idx = 0`. Go cannot express field
// defaults; the zero value of each field is the same state.
type SGRParser struct {
	params     []uint16
	params_sep ParserActionCSISepList
	// Zig: usize.
	idx uint
}

// SGRParserEmpty is sgr.Parser.empty: empty state parser.
//
// Zig: `.{}`, every field at its default. The zero value is already that
// state.
var SGRParserEmpty = SGRParser{}

// Next is sgr.Parser.next: next returns the next attribute or null if there
// are no more attributes.
func (self *SGRParser) Next() Optional[SGRAttribute] {
	if self.idx >= uint(len(self.params)) {
		// We're more likely to not be done than to be done.

		// Add one to ensure we don't loop on unset
		defer func() { self.idx += 1 }()

		// If we're at index zero it means we must have an empty list
		// and an empty list implicitly means unset, otherwise we're
		// done and return null.
		if self.idx == 0 {
			return Some(SGRAttribute{Tag: SGRAttributeTagUnset})
		}
		return None[SGRAttribute]()
	}

	slice := self.params[self.idx:len(self.params)]
	// Call inlined for performance reasons.
	colon := self.params_sep.isSet(int(self.idx))
	self.idx += 1

	// Our last one will have an idx be the last value.
	if len(slice) == 0 {
		return None[SGRAttribute]()
	}

	// If we have a colon separator then we need to ensure we're
	// parsing a value that allows it.
	if colon {
		// Colons are fairly rare in the wild.

		switch slice[0] {
		case 4, 38, 48, 58:

		default:
			// In real world use it's very rare
			// that we receive an invalid sequence.

			// Consume all the colon separated
			// values and return them as unknown.
			start := self.idx
			for self.params_sep.isSet(int(self.idx)) {
				self.idx += 1
			}
			self.idx += 1
			return Some(SGRAttribute{Tag: SGRAttributeTagUnknown, Unknown: SGRAttributeUnknown{
				full:    self.params,
				partial: slice[0:min(self.idx-start+1, uint(len(slice)))],
			}})
		}
	}

	switch slice[0] {
	case 0:
		return Some(SGRAttribute{Tag: SGRAttributeTagUnset})

	case 1:
		return Some(SGRAttribute{Tag: SGRAttributeTagBold})

	case 2:
		return Some(SGRAttribute{Tag: SGRAttributeTagFaint})

	case 3:
		return Some(SGRAttribute{Tag: SGRAttributeTagItalic})

	case 4:
		// Zig: a block labelled `underline`; `break :underline` leaves it and
		// the switch, which a plain Go `break` does.
		if colon {
			// Colons are fairly rare in the wild.

			// A trailing colon with no following sub-param
			// (e.g. "ESC[58:4:m") leaves the colon separator
			// bit set on the last param without adding another
			// entry, so we can see param 4 with a colon but
			// nothing after it.
			if len(slice) < 2 {
				break
			}

			if self.isColon() {
				// Invalid/unknown SGRs are just not very likely.

				self.consumeUnknownColon()
				break
			}

			self.idx += 1
			var u SGRAttributeUnderline
			switch slice[1] {
			case 0:
				u = SGRAttributeUnderlineNone
			case 1:
				u = SGRAttributeUnderlineSingle
			case 2:
				u = SGRAttributeUnderlineDouble
			case 3:
				u = SGRAttributeUnderlineCurly
			case 4:
				u = SGRAttributeUnderlineDotted
			case 5:
				u = SGRAttributeUnderlineDashed

			// For unknown underline styles,
			// just render a single underline.
			default:
				// This is quite a rare condition.
				u = SGRAttributeUnderlineSingle
			}
			return Some(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: u})
		}

		return Some(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle})

	case 5:
		return Some(SGRAttribute{Tag: SGRAttributeTagBlink})

	case 6:
		return Some(SGRAttribute{Tag: SGRAttributeTagBlink})

	case 7:
		return Some(SGRAttribute{Tag: SGRAttributeTagInverse})

	case 8:
		return Some(SGRAttribute{Tag: SGRAttributeTagInvisible})

	case 9:
		return Some(SGRAttribute{Tag: SGRAttributeTagStrikethrough})

	case 21:
		return Some(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineDouble})

	case 22:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetBold})

	case 23:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetItalic})

	case 24:
		return Some(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineNone})

	case 25:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetBlink})

	case 27:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetInverse})

	case 28:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetInvisible})

	case 29:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetStrikethrough})

	case 30, 31, 32, 33, 34, 35, 36, 37:
		return Some(SGRAttribute{
			Tag:  SGRAttributeTag8Fg,
			X8Fg: ColorName(slice[0] - 30),
		})

	case 38:
		if len(slice) >= 2 {
			// We are very likely to have enough parameters.

			switch slice[1] {
			// `2` indicates direct-color (r, g, b).
			// We need at least 3 more params for this to make sense.
			case 2:
				if v := self.parseDirectColor(
					SGRAttributeTagDirectColorFg,
					slice,
					colon,
				); v.Valid {
					return v
				}

			// `5` indicates indexed color.
			case 5:
				if len(slice) >= 3 {
					// We are very likely to have enough parameters.

					self.idx += 2
					return Some(SGRAttribute{
						Tag: SGRAttributeTag256Fg,
						// Zig: @truncate to u8.
						X256Fg: uint8(slice[2]),
					})
				}

			default:
			}
		}

	case 39:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetFg})

	case 40, 41, 42, 43, 44, 45, 46, 47:
		return Some(SGRAttribute{
			Tag:  SGRAttributeTag8Bg,
			X8Bg: ColorName(slice[0] - 40),
		})

	case 48:
		if len(slice) >= 2 {
			// We are very likely to have enough parameters.

			switch slice[1] {
			// `2` indicates direct-color (r, g, b).
			// We need at least 3 more params for this to make sense.
			case 2:
				if v := self.parseDirectColor(
					SGRAttributeTagDirectColorBg,
					slice,
					colon,
				); v.Valid {
					return v
				}

			// `5` indicates indexed color.
			case 5:
				if len(slice) >= 3 {
					// We are very likely to have enough parameters.

					self.idx += 2
					return Some(SGRAttribute{
						Tag: SGRAttributeTag256Bg,
						// Zig: @truncate to u8.
						X256Bg: uint8(slice[2]),
					})
				}

			default:
			}
		}

	case 49:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetBg})

	case 53:
		return Some(SGRAttribute{Tag: SGRAttributeTagOverline})
	case 55:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetOverline})

	case 58:
		if len(slice) >= 2 {
			// We are very likely to have enough parameters.

			switch slice[1] {
			// `2` indicates direct-color (r, g, b).
			// We need at least 3 more params for this to make sense.
			case 2:
				if v := self.parseDirectColor(
					SGRAttributeTagUnderlineColor,
					slice,
					colon,
				); v.Valid {
					return v
				}

			// `5` indicates indexed color.
			case 5:
				if len(slice) >= 3 {
					// We are very likely to have enough parameters.

					self.idx += 2
					return Some(SGRAttribute{
						Tag: SGRAttributeTag256UnderlineColor,
						// Zig: @truncate to u8.
						X256UnderlineColor: uint8(slice[2]),
					})
				}
			default:
			}
		}

	case 59:
		return Some(SGRAttribute{Tag: SGRAttributeTagResetUnderlineColor})

	case 90, 91, 92, 93, 94, 95, 96, 97:
		return Some(SGRAttribute{
			Tag: SGRAttributeTag8BrightFg,
			// 82 instead of 90 to offset to "bright" colors
			X8BrightFg: ColorName(slice[0] - 82),
		})

	case 100, 101, 102, 103, 104, 105, 106, 107:
		return Some(SGRAttribute{
			Tag:        SGRAttributeTag8BrightBg,
			X8BrightBg: ColorName(slice[0] - 92),
		})

	default:
	}

	return Some(SGRAttribute{Tag: SGRAttributeTagUnknown, Unknown: SGRAttributeUnknown{full: self.params, partial: slice}})
}

// parseDirectColor is sgr.Parser.parseDirectColor.
//
// Zig: `tag` is a comptime parameter naming the union variant to build
// (@unionInit); here it is a run-time value, and sgrAttributeInitRGB sets
// the payload field it names.
func (self *SGRParser) parseDirectColor(
	tag SGRAttributeTag,
	slice []uint16,
	colon bool,
) Optional[SGRAttribute] {
	// Any direct color style must have at least 5 values.
	if len(slice) < 5 {
		return None[SGRAttribute]()
	}

	// Only used for direct color sets (38, 48, 58) and subparam 2.
	assert(slice[1] == 2)

	// Note: We use @truncate because the value should be 0 to 255. If
	// it isn't, the behavior is undefined so we just... truncate it.

	// If we don't have a colon, then we expect exactly 3 semicolon
	// separated values.
	if !colon {
		// Semicolons are much more common than colons.

		self.idx += 4
		return Some(sgrAttributeInitRGB(tag, ColorRGB{
			r: uint8(slice[2]),
			g: uint8(slice[3]),
			b: uint8(slice[4]),
		}))
	}

	// We have a colon, we might have either 5 or 6 values depending
	// on if the colorspace is present.
	count := self.countColon()
	switch count {
	case 3:
		// This is the much more common case in the wild.

		self.idx += 4
		return Some(sgrAttributeInitRGB(tag, ColorRGB{
			r: uint8(slice[2]),
			g: uint8(slice[3]),
			b: uint8(slice[4]),
		}))

	case 4:
		self.idx += 5
		return Some(sgrAttributeInitRGB(tag, ColorRGB{
			r: uint8(slice[3]),
			g: uint8(slice[4]),
			b: uint8(slice[5]),
		}))

	default:
		// Invalid/unknown SGRs just don't happen very often at all.

		self.consumeUnknownColon()
		return None[SGRAttribute]()
	}
}

// sgrAttributeInitRGB is `@unionInit(Attribute, @tagName(tag), rgb)` for the
// three variants whose payload is an RGB. Not a Zig function: the Go stand-in
// for the comptime @unionInit in parseDirectColor.
func sgrAttributeInitRGB(tag SGRAttributeTag, rgb ColorRGB) SGRAttribute {
	switch tag {
	case SGRAttributeTagDirectColorFg:
		return SGRAttribute{Tag: tag, DirectColorFg: rgb}
	case SGRAttributeTagDirectColorBg:
		return SGRAttribute{Tag: tag, DirectColorBg: rgb}
	case SGRAttributeTagUnderlineColor:
		return SGRAttribute{Tag: tag, UnderlineColor: rgb}
	}
	// Zig: a compile error for any other tag.
	panic("unreachable")
}

// isColon is sgr.Parser.isColon: returns true if the present position has a
// colon separator. This always returns false for the last value since it has
// no separator.
func (self *SGRParser) isColon() bool {
	// Call inlined for performance reasons.
	return self.params_sep.isSet(int(self.idx))
}

// countColon is sgr.Parser.countColon.
//
// Zig: returns usize.
func (self *SGRParser) countColon() uint {
	var count uint = 0
	idx := self.idx
	// Zig: `self.params.len - 1` traps on an empty list. Every caller has
	// already read params[idx-1], so the list is never empty here.
	assert(len(self.params) > 0)
	for idx < uint(len(self.params))-1 && self.params_sep.isSet(int(idx)) {
		count += 1
		idx += 1
	}
	return count
}

// consumeUnknownColon is sgr.Parser.consumeUnknownColon: consumes all the
// remaining parameters separated by a colon and returns an unknown
// attribute.
func (self *SGRParser) consumeUnknownColon() {
	count := self.countColon()
	self.idx += count + 1
}
