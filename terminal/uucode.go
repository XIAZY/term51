// The two types of the uucode package (jacobsandlund/uucode 0.2.0, the
// version pinned in Ghostty v1.3.1's build.zig.zon) that libghostty-vt
// names: uucode.grapheme.BreakState (src/grapheme.zig) and
// uucode.x.types.GraphemeBreakNoControl (src/x/types_x/grapheme.zig,
// re-exported by src/x/types.x.zig). Only the types are ported, with
// uucode's numeric values: the values are the bit patterns stored in
// Precompute.Key / Precompute.Value (unicode_grapheme.go) and in the
// Properties entries of the generated tables (unicode_props.go), which
// are embedded byte for byte.
//
// uucode's data generation and break algorithm
// (uucode.x.grapheme.computeGraphemeBreakNoControl) are not ported (project
// lead decision): their results are the embedded tables, dumped from the
// pinned Zig build by term51's tools/dump_unicode.sh.
//
// Names follow the Zig path the callers write: uucode.grapheme.BreakState
// is UucodeGraphemeBreakState, uucode.x.types.GraphemeBreakNoControl is
// UucodeXTypesGraphemeBreakNoControl; constants are the type name plus the
// tag in Go case.

package terminal

// UucodeGraphemeBreakState is uucode.grapheme.BreakState.
//
// Zig: enum(u3).
type UucodeGraphemeBreakState uint8

const (
	UucodeGraphemeBreakStateDefault UucodeGraphemeBreakState = iota
	UucodeGraphemeBreakStateRegionalIndicator
	UucodeGraphemeBreakStateExtendedPictographic
	UucodeGraphemeBreakStateIndicConjunctBreakConsonant
	UucodeGraphemeBreakStateIndicConjunctBreakLinker
)

// uucodeGraphemeBreakStateNames are the tag names of
// UucodeGraphemeBreakState, in order.
var uucodeGraphemeBreakStateNames = [...]string{
	"default",
	"regional_indicator",
	"extended_pictographic",
	"indic_conjunct_break_consonant",
	"indic_conjunct_break_linker",
}

// String returns the Zig tag name (@tagName).
func (self UucodeGraphemeBreakState) String() string {
	return uucodeGraphemeBreakStateNames[self]
}

// UucodeXTypesGraphemeBreakNoControl is
// uucode.x.types.GraphemeBreakNoControl.
//
// Zig: enum(u5).
type UucodeXTypesGraphemeBreakNoControl uint8

const (
	UucodeXTypesGraphemeBreakNoControlOther UucodeXTypesGraphemeBreakNoControl = iota
	UucodeXTypesGraphemeBreakNoControlPrepend
	UucodeXTypesGraphemeBreakNoControlRegionalIndicator
	UucodeXTypesGraphemeBreakNoControlSpacingMark
	UucodeXTypesGraphemeBreakNoControlL
	UucodeXTypesGraphemeBreakNoControlV
	UucodeXTypesGraphemeBreakNoControlT
	UucodeXTypesGraphemeBreakNoControlLv
	UucodeXTypesGraphemeBreakNoControlLvt
	UucodeXTypesGraphemeBreakNoControlZwj
	UucodeXTypesGraphemeBreakNoControlZwnj
	UucodeXTypesGraphemeBreakNoControlExtendedPictographic
	UucodeXTypesGraphemeBreakNoControlEmojiModifierBase
	UucodeXTypesGraphemeBreakNoControlEmojiModifier
	// extend, ==
	//   zwnj +
	//   indic_conjunct_break_extend +
	//   indic_conjunct_break_linker
	UucodeXTypesGraphemeBreakNoControlIndicConjunctBreakExtend
	UucodeXTypesGraphemeBreakNoControlIndicConjunctBreakLinker
	UucodeXTypesGraphemeBreakNoControlIndicConjunctBreakConsonant
)

// uucodeXTypesGraphemeBreakNoControlNames are the tag names of
// UucodeXTypesGraphemeBreakNoControl, in order.
var uucodeXTypesGraphemeBreakNoControlNames = [...]string{
	"other",
	"prepend",
	"regional_indicator",
	"spacing_mark",
	"l",
	"v",
	"t",
	"lv",
	"lvt",
	"zwj",
	"zwnj",
	"extended_pictographic",
	"emoji_modifier_base",
	"emoji_modifier",
	"indic_conjunct_break_extend",
	"indic_conjunct_break_linker",
	"indic_conjunct_break_consonant",
}

// String returns the Zig tag name (@tagName).
func (self UucodeXTypesGraphemeBreakNoControl) String() string {
	return uucodeXTypesGraphemeBreakNoControlNames[self]
}
