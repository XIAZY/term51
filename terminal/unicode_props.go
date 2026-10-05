// Port of src/unicode/props.zig (Ghostty v1.3.1).
//
// Property set per codepoint that Ghostty cares about.
//
// Adding to this lets you find new properties but also potentially makes
// our lookup tables less efficient. Any changes to this should run the
// benchmarks in src/bench to verify that we haven't regressed.
//
// Naming: callers reach the type through the unicode namespace
// (unicode/main.zig: `pub const Properties = @import("props.zig")
// .Properties`), so it is `UnicodeProperties`, the namespace name, as
// `color.RGB` is `ColorRGB`. Fields keep Zig's names.
//
// The packed struct's bit layout is observable: the stage3 entries of the
// embedded tables (unicode_props_table.go) are its bit patterns, so the
// conversion pair Int / UnicodePropertiesFromInt places each field at
// Zig's bit offset (mapping.md, packed structs).

package terminal

// UnicodeProperties is unicode.Properties.
//
// Zig: `packed struct` (backing integer u9): width is bits 0..1,
// width_zero_in_grapheme bit 2, grapheme_break bits 3..7, emoji_vs_base
// bit 8. The Go zero value is Zig's default (0, false, .other, false).
type UnicodeProperties struct {
	// Codepoint width. We clamp to [0, 2] since Ghostty handles control
	// characters and we max out at 2 for wide characters (i.e. 3-em dash
	// becomes a 2-em dash).
	//
	// Zig: u2.
	width uint8

	// Whether the code point does not contribute to the width of a grapheme
	// cluster (not used for single code point cells).
	width_zero_in_grapheme bool

	// Grapheme break property.
	grapheme_break UucodeXTypesGraphemeBreakNoControl

	// Emoji VS compatibility
	emoji_vs_base bool
}

// Eql reports whether two Properties are equal.
//
// Needed for lut.Generator
func (a UnicodeProperties) Eql(b UnicodeProperties) bool {
	return a.width == b.width &&
		a.width_zero_in_grapheme == b.width_zero_in_grapheme &&
		a.grapheme_break == b.grapheme_break &&
		a.emoji_vs_base == b.emoji_vs_base
}

// Format is Properties.format: the Zig literal of the value, as
// lut.Tables.writeZig writes each stage3 entry (`{f}`).
//
// Needed for lut.Generator
func (self UnicodeProperties) Format() string {
	return ".{\n" +
		"    .width= " + itoa(uint64(self.width)) + ",\n" +
		"    .width_zero_in_grapheme= " + unicodePropsBool(self.width_zero_in_grapheme) + ",\n" +
		"    .grapheme_break= ." + self.grapheme_break.String() + ",\n" +
		"    .emoji_vs_base= " + unicodePropsBool(self.emoji_vs_base) + ",\n" +
		"}"
}

// unicodePropsBool is Zig's `{}` on a bool.
func unicodePropsBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Int is @bitCast(self) to the backing integer (u9; the Go result is
// always below 0x200).
func (self UnicodeProperties) Int() uint16 {
	assert(self.width <= 3)
	assert(self.grapheme_break <= 0x1F)
	v := uint16(self.width) | uint16(self.grapheme_break)<<3
	if self.width_zero_in_grapheme {
		v |= 1 << 2
	}
	if self.emoji_vs_base {
		v |= 1 << 8
	}
	return v
}

// UnicodePropertiesFromInt is @bitCast(v) from the backing integer u9 to
// Properties. v must fit in 9 bits.
func UnicodePropertiesFromInt(v uint16) UnicodeProperties {
	assert(v <= 0x1FF)
	return UnicodeProperties{
		width:                  uint8(v & 0x3),
		width_zero_in_grapheme: v&(1<<2) != 0,
		grapheme_break:         UucodeXTypesGraphemeBreakNoControl(v >> 3 & 0x1F),
		emoji_vs_base:          v&(1<<8) != 0,
	}
}
