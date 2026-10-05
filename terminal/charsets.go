// Port of src/terminal/charsets.zig (Ghostty v1.3.1).
//
// File naming: charsets.zig is a namespace file, so its declarations take
// the prefix `Charsets` (`charsets` for private ones).
//
// Declarations appear in the same order as in charsets.zig. The C ABI
// selection (the C variant of every `LibEnum`) is not ported; see
// docs/mapping.md. A `LibEnum` with the Zig target is an ordinary enum whose
// values are numbered from zero in the order listed.
//
// The three tables are computed at compile time in Zig; in Go they are
// package variables initialised once at start-up (docs/mapping.md,
// "comptime table"), each by a function holding the Zig block. `table` is
// `inline` in Zig; Go leaves inlining to the compiler.

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

// charsetsSlotsNames are the tag names of CharsetsSlots, in order.
var charsetsSlotsNames = [...]string{"G0", "G1", "G2", "G3"}

// String returns the Zig tag name (@tagName).
func (self CharsetsSlots) String() string { return charsetsSlotsNames[self] }

// CharsetsActiveSlot is charsets.ActiveSlot: the name of the active slots.
type CharsetsActiveSlot uint8

const (
	CharsetsActiveSlotGL CharsetsActiveSlot = iota
	CharsetsActiveSlotGR
)

// charsetsActiveSlotNames are the tag names of CharsetsActiveSlot, in order.
var charsetsActiveSlotNames = [...]string{"GL", "GR"}

// String returns the Zig tag name (@tagName).
func (self CharsetsActiveSlot) String() string { return charsetsActiveSlotNames[self] }

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

// charsetsCharsetNames are the tag names of CharsetsCharset, in order.
var charsetsCharsetNames = [charsetsCharsetCount]string{"utf8", "ascii", "british", "dec_special"}

// String returns the Zig tag name (@tagName).
func (self CharsetsCharset) String() string { return charsetsCharsetNames[self] }

// CharsetsTable is charsets.table: the table for the given charset. This
// returns a pointer to a slice that is guaranteed to be 255 chars that can
// be used to map ASCII to the given charset.
//
// (Upstream's comment says 255; the tables have table_len = 256 entries.)
// The returned slice aliases the package table, as Zig's points at the
// comptime constant; callers must not write through it.
func CharsetsTable(set CharsetsCharset) []uint16 {
	switch set {
	case CharsetsCharsetBritish:
		return charsetsBritish[:]
	case CharsetsCharsetDecSpecial:
		return charsetsDecSpecial[:]

	// utf8 is not a table, callers should double-check if the
	// charset is utf8 and NOT use tables.
	case CharsetsCharsetUtf8:
		panic("unreachable")

	// recommended that callers just map ascii directly but we can
	// support a table
	case CharsetsCharsetAscii:
		return charsetsAscii[:]
	}
	// Zig's switch is exhaustive; a value outside the enum traps.
	panic("unreachable")
}

// charsetsAscii is charsets.ascii: just a basic c => c ascii table.
//
// Zig: initTable(), at compile time.
var charsetsAscii = charsetsInitTable()

// charsetsBritish is charsets.british.
// https://vt100.net/docs/vt220-rm/chapter2.html
//
// Zig: the `british:` block, at compile time.
var charsetsBritish = func() [charsetsTableLen]uint16 {
	tbl := charsetsInitTable()
	tbl[0x23] = 0x00a3
	return tbl
}()

// charsetsDecSpecial is charsets.dec_special.
// https://en.wikipedia.org/wiki/DEC_Special_Graphics
//
// Zig: the `tech:` block, at compile time.
var charsetsDecSpecial = func() [charsetsTableLen]uint16 {
	tbl := charsetsInitTable()
	tbl[0x60] = 0x25C6
	tbl[0x61] = 0x2592
	tbl[0x62] = 0x2409
	tbl[0x63] = 0x240C
	tbl[0x64] = 0x240D
	tbl[0x65] = 0x240A
	tbl[0x66] = 0x00B0
	tbl[0x67] = 0x00B1
	tbl[0x68] = 0x2424
	tbl[0x69] = 0x240B
	tbl[0x6a] = 0x2518
	tbl[0x6b] = 0x2510
	tbl[0x6c] = 0x250C
	tbl[0x6d] = 0x2514
	tbl[0x6e] = 0x253C
	tbl[0x6f] = 0x23BA
	tbl[0x70] = 0x23BB
	tbl[0x71] = 0x2500
	tbl[0x72] = 0x23BC
	tbl[0x73] = 0x23BD
	tbl[0x74] = 0x251C
	tbl[0x75] = 0x2524
	tbl[0x76] = 0x2534
	tbl[0x77] = 0x252C
	tbl[0x78] = 0x2502
	tbl[0x79] = 0x2264
	tbl[0x7a] = 0x2265
	tbl[0x7b] = 0x03C0
	tbl[0x7c] = 0x2260
	tbl[0x7d] = 0x00A3
	tbl[0x7e] = 0x00B7
	return tbl
}()

// charsetsTableLen is charsets.table_len: our table length is 256 so we can
// contain all ASCII chars.
//
// Zig: std.math.maxInt(u8) + 1.
const charsetsTableLen = 255 + 1

// charsetsInitTable is charsets.initTable: creates a table that maps ASCII
// to ASCII as a getting started point.
func charsetsInitTable() [charsetsTableLen]uint16 {
	var result [charsetsTableLen]uint16
	var i int = 0
	for ; i < charsetsTableLen; i += 1 {
		result[i] = uint16(i)
	}
	assert(i == charsetsTableLen)
	return result
}
