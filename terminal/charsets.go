// WIREFRAME for src/terminal/charsets.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then. See the header of
// color.go for the rules.
//
// File naming: charsets.zig is a namespace file, so its declarations take
// the prefix `Charsets` (`charsets` for private ones).
//
// Declarations appear in the same order as in charsets.zig. The C ABI
// selection (the C variant of every `LibEnum`) is not ported; see
// docs/mapping.md. A `LibEnum` with the Zig target is an ordinary enum whose
// values are numbered from zero in the order listed.

package terminal

// CharsetsSlots is charsets.Slots: the available charset slots for a
// terminal.
type CharsetsSlots uint8

const (
	CharsetsSlotsG0 CharsetsSlots = iota
	CharsetsSlotsG1
	CharsetsSlotsG2
	CharsetsSlotsG3
)

// String returns the Zig tag name (@tagName).
func (self CharsetsSlots) String() string { panic(notImplemented) }

// CharsetsActiveSlot is charsets.ActiveSlot: the name of the active slots.
type CharsetsActiveSlot uint8

const (
	CharsetsActiveSlotGL CharsetsActiveSlot = iota
	CharsetsActiveSlotGR
)

// String returns the Zig tag name (@tagName).
func (self CharsetsActiveSlot) String() string { panic(notImplemented) }

// CharsetsCharset is charsets.Charset: the list of supported character sets
// and their associated tables.
type CharsetsCharset uint8

const (
	CharsetsCharsetUtf8 CharsetsCharset = iota
	CharsetsCharsetAscii
	CharsetsCharsetBritish
	CharsetsCharsetDecSpecial

	charsetsCharsetCount = iota
)

// String returns the Zig tag name (@tagName).
func (self CharsetsCharset) String() string { panic(notImplemented) }

// CharsetsTable is charsets.table: the table for the given charset. This
// returns a slice that is guaranteed to be 256 chars that can be used to map
// ASCII to the given charset.
//
// Zig: `.utf8` is `unreachable`; utf8 is not a table, callers should
// double-check if the charset is utf8 and NOT use tables.
func CharsetsTable(set CharsetsCharset) []uint16 { panic(notImplemented) }

// charsetsAscii is charsets.ascii: just a basic c => c ascii table.
//
// Zig: initTable(), at compile time. Wireframe: left zero.
var charsetsAscii [charsetsTableLen]uint16

// charsetsBritish is charsets.british.
// https://vt100.net/docs/vt220-rm/chapter2.html
//
// Zig: initTable() with 0x23 replaced, at compile time. Wireframe: left
// zero.
var charsetsBritish [charsetsTableLen]uint16

// charsetsDecSpecial is charsets.dec_special.
// https://en.wikipedia.org/wiki/DEC_Special_Graphics
//
// Zig: initTable() with 0x60 to 0x7e replaced, at compile time. Wireframe:
// left zero.
var charsetsDecSpecial [charsetsTableLen]uint16

// charsetsTableLen is charsets.table_len: our table length is 256 so we can
// contain all ASCII chars.
//
// Zig: std.math.maxInt(u8) + 1.
const charsetsTableLen = 255 + 1

// charsetsInitTable is charsets.initTable: creates a table that maps ASCII
// to ASCII as a getting started point.
func charsetsInitTable() [charsetsTableLen]uint16 { panic(notImplemented) }
