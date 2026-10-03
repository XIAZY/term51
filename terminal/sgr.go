// WIREFRAME for src/terminal/sgr.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// SGR (Select Graphic Rendition) attribute parsing and types.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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

// String returns the Zig tag name (@tagName).
func (self SGRAttributeTag) String() string { panic(notImplemented) }

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

// String returns the Zig tag name (@tagName).
func (self SGRAttributeUnderline) String() string { panic(notImplemented) }

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
func (self *SGRParser) Next() Optional[SGRAttribute] { panic(notImplemented) }

// parseDirectColor is sgr.Parser.parseDirectColor.
//
// Zig: `tag` is a comptime parameter naming the union variant to build.
func (self *SGRParser) parseDirectColor(
	tag SGRAttributeTag,
	slice []uint16,
	colon bool,
) Optional[SGRAttribute] {
	panic(notImplemented)
}

// isColon is sgr.Parser.isColon: returns true if the present position has a
// colon separator. This always returns false for the last value since it has
// no separator.
func (self *SGRParser) isColon() bool { panic(notImplemented) }

// countColon is sgr.Parser.countColon.
//
// Zig: returns usize.
func (self *SGRParser) countColon() uint { panic(notImplemented) }

// consumeUnknownColon is sgr.Parser.consumeUnknownColon: consumes all the
// remaining parameters separated by a colon and returns an unknown
// attribute.
func (self *SGRParser) consumeUnknownColon() { panic(notImplemented) }
