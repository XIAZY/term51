// Port of src/unicode/grapheme.zig (Ghostty v1.3.1).
//
// Naming: callers use graphemeBreak through the unicode namespace
// (unicode/main.zig: `pub const graphemeBreak = grapheme.graphemeBreak`), so
// it is `UnicodeGraphemeBreak`, the namespace name. The private namespace
// struct `Precompute` and its declarations take the file prefix in lower
// case: `unicodeGraphemePrecomputeKey`, `unicodeGraphemePrecomputeValue`,
// `unicodeGraphemePrecomputeData`.
//
// Precompute.data: Zig computes the 8,192-entry table at compile time by
// calling uucode.x.grapheme.computeGraphemeBreakNoControl for every
// (state, gb1, gb2) built from valid enum values. uucode's algorithm is not
// ported (project lead decision): res/grapheme_precompute.bin is the
// table's 8,192 bytes as the pinned Zig build has them in memory (one
// Value, packed struct(u4), per byte), dumped by term51's
// tools/dump_unicode.sh (gated test "term51: graphemeBreak: dump exact data
// when TERM51_UNICODE_DUMP_DIR is set" in grapheme.zig) and embedded byte
// for byte. The 6,747 entries whose index is not built from valid enum
// values are `undefined` in Zig (`var result: [...]Value = undefined`); the
// build emits them as zero bytes, which the Go table keeps. They cannot be
// read: Key's fields are always valid enum values.
//
// Not ported: `main` (a verification binary comparing graphemeBreak with
// uucode.grapheme.isBreak over every codepoint pair; it needs uucode's
// data and algorithm, and is not part of the library) and `std_options`
// (its log level).

package terminal

import _ "embed"

// UnicodeGraphemeBreak is unicode.graphemeBreak: determines if there is a
// grapheme break between two codepoints. This must be called sequentially
// maintaining the state between calls.
//
// This function does NOT work with control characters. Control characters,
// line feeds, and carriage returns are expected to be filtered out before
// calling this function. This is because this function is tuned for
// Ghostty.
//
// Zig: cp1 and cp2 are u21 (UnicodeTable.Get asserts the range).
func UnicodeGraphemeBreak(cp1 uint32, cp2 uint32, state *UucodeGraphemeBreakState) bool {
	value := unicodeGraphemePrecomputeData[(unicodeGraphemePrecomputeKey{
		gb1:   UnicodeTable.Get(cp1).grapheme_break,
		gb2:   UnicodeTable.Get(cp2).grapheme_break,
		state: *state,
	}).index()]
	*state = value.state
	return value.result
}

// This is all the structures and data for the precomputed lookup table
// for all possible permutations of state and grapheme break properties.
// Precomputation requires 2^13 keys of 4 bit values so the whole table is
// 8KB.

// unicodeGraphemePrecomputeKey is Precompute.Key.
//
// Zig: packed struct(u13): state is bits 0..2, gb1 bits 3..7, gb2 bits
// 8..12.
type unicodeGraphemePrecomputeKey struct {
	state UucodeGraphemeBreakState
	gb1   UucodeXTypesGraphemeBreakNoControl
	gb2   UucodeXTypesGraphemeBreakNoControl
}

// index is Key.index: the key's u13 bit pattern (@bitCast) as an index.
// The fields are a u3 and two u5 enums in Zig; a wider value cannot exist
// there, so the port asserts that each fits instead of dropping bits.
func (self unicodeGraphemePrecomputeKey) index() uint {
	assert(self.state <= 0x7)
	assert(self.gb1 <= 0x1F)
	assert(self.gb2 <= 0x1F)
	return uint(self.state) | uint(self.gb1)<<3 | uint(self.gb2)<<8
}

// unicodeGraphemePrecomputeValue is Precompute.Value.
//
// Zig: packed struct(u4): result is bit 0, state bits 1..3.
type unicodeGraphemePrecomputeValue struct {
	result bool
	state  UucodeGraphemeBreakState
}

// unicodeGraphemePrecomputeValueFromInt is @bitCast(v) from the backing
// integer u4 to Value. v must fit in 4 bits.
func unicodeGraphemePrecomputeValueFromInt(v uint8) unicodeGraphemePrecomputeValue {
	assert(v <= 0xF)
	return unicodeGraphemePrecomputeValue{
		result: v&1 != 0,
		state:  UucodeGraphemeBreakState(v >> 1),
	}
}

// unicodeGraphemePrecomputeBin is res/grapheme_precompute.bin (see the file
// comment).
//
//go:embed res/grapheme_precompute.bin
var unicodeGraphemePrecomputeBin []byte

// unicodeGraphemePrecomputeData is Precompute.data.
var unicodeGraphemePrecomputeData = unicodeGraphemePrecomputeInit(unicodeGraphemePrecomputeBin)

// unicodeGraphemePrecomputeInit decodes res/grapheme_precompute.bin into
// Precompute.data, once at start-up (Zig: a comptime block). A malformed
// file stops the program at start-up.
func unicodeGraphemePrecomputeInit(data []byte) [8192]unicodeGraphemePrecomputeValue {
	var result [8192]unicodeGraphemePrecomputeValue
	// Zig: std.debug.assert(@sizeOf(@TypeOf(result)) == 8192)
	if len(data) != len(result) {
		panic("unicode_grapheme: res/grapheme_precompute.bin is not 8192 bytes")
	}
	for i, b := range data {
		if b > 0xF {
			panic("unicode_grapheme: invalid Value in res/grapheme_precompute.bin")
		}
		result[i] = unicodeGraphemePrecomputeValueFromInt(b)
	}
	return result
}
