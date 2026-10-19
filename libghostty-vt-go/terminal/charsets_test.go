// Tests translated one for one from src/terminal/charsets.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order (helpers in Parser_test.go).
//   - `@typeInfo(E).@"enum".fields` has no Go counterpart: the field names
//     are the String() of each value, and the field count is one more than
//     the last value (the values are numbered from zero, in order).
//   - `@intFromPtr(table(x).ptr)` is the address of the first element of the
//     returned slice.

package terminal

import (
	"reflect"
	"testing"
)

// Zig: test (unnamed, charsets.zig:101)
func TestCharsets_unnamed(t *testing.T) {
	// Zig: inline for over the Charset fields.
	for c := CharsetsCharset(0); c < charsetsCharsetCount; c++ {
		// utf8 has no table
		if c == CharsetsCharsetUtf8 {
			continue
		}

		tbl := CharsetsTable(c)

		// Yes, I could use `table_len` here, but I want to explicitly use a
		// hardcoded constant so that if there are miscompilations or a comptime
		// issue, we catch it.
		expectEqual(t, 256, len(tbl))
	}
}

// Zig: test "term51: Slots: names and order"
func TestCharsets_term51_Slots_names_and_order(t *testing.T) {
	expectEqual(t, 4, int(CharsetsSlotsG3)+1) // fields.len
	expectEqualStrings(t, "G0", CharsetsSlots(0).String())
	expectEqualStrings(t, "G1", CharsetsSlots(1).String())
	expectEqualStrings(t, "G2", CharsetsSlots(2).String())
	expectEqualStrings(t, "G3", CharsetsSlots(3).String())
	expectEqual(t, 0, int(CharsetsSlotsG0))
	expectEqual(t, 3, int(CharsetsSlotsG3))
}

// Zig: test "term51: ActiveSlot: names and order"
func TestCharsets_term51_ActiveSlot_names_and_order(t *testing.T) {
	expectEqual(t, 2, int(CharsetsActiveSlotGR)+1) // fields.len
	expectEqualStrings(t, "GL", CharsetsActiveSlot(0).String())
	expectEqualStrings(t, "GR", CharsetsActiveSlot(1).String())
	expectEqual(t, 0, int(CharsetsActiveSlotGL))
	expectEqual(t, 1, int(CharsetsActiveSlotGR))
}

// Zig: test "term51: Charset: names and order"
func TestCharsets_term51_Charset_names_and_order(t *testing.T) {
	expectEqual(t, 4, int(charsetsCharsetCount)) // fields.len
	expectEqualStrings(t, "utf8", CharsetsCharset(0).String())
	expectEqualStrings(t, "ascii", CharsetsCharset(1).String())
	expectEqualStrings(t, "british", CharsetsCharset(2).String())
	expectEqualStrings(t, "dec_special", CharsetsCharset(3).String())
	expectEqual(t, 0, int(CharsetsCharsetUtf8))
	expectEqual(t, 1, int(CharsetsCharsetAscii))
	expectEqual(t, 2, int(CharsetsCharsetBritish))
	expectEqual(t, 3, int(CharsetsCharsetDecSpecial))
}

// Zig: test "term51: initTable: identity over all 256 entries"
func TestCharsets_term51_initTable_identity_over_all_256_entries(t *testing.T) {
	tbl := charsetsInitTable()
	expectEqual(t, 256, len(tbl))
	expect(t, reflect.TypeOf(tbl) == reflect.TypeOf([256]uint16{}), "@TypeOf(tbl) == [256]u16")
	for i, v := range tbl {
		expectEqual(t, uint16(i), v)
	}
	expectEqual(t, uint16(0), tbl[0])
	expectEqual(t, uint16(255), tbl[255])
}

// Zig: test "term51: table: ascii is the identity map"
func TestCharsets_term51_table_ascii_is_the_identity_map(t *testing.T) {
	tbl := CharsetsTable(CharsetsCharsetAscii)
	expectEqual(t, 256, len(tbl))
	for i, v := range tbl {
		expectEqual(t, uint16(i), v)
	}
}

// Zig: test "term51: table: british only remaps 0x23 to the pound sign"
func TestCharsets_term51_table_british_only_remaps_0x23_to_the_pound_sign(t *testing.T) {
	tbl := CharsetsTable(CharsetsCharsetBritish)
	expectEqual(t, 256, len(tbl))
	for i, v := range tbl {
		if i == 0x23 {
			expectEqual(t, uint16(0x00A3), v)
		} else {
			expectEqual(t, uint16(i), v)
		}
	}
}

// Zig: test "term51: table: dec_special remaps exactly 0x60 through 0x7E"
func TestCharsets_term51_table_dec_special_remaps_exactly_0x60_through_0x7E(t *testing.T) {
	tbl := CharsetsTable(CharsetsCharsetDecSpecial)
	expectEqual(t, 256, len(tbl))

	// Independent copy of the DEC Special Graphics mapping for 0x60..0x7E.
	expected := [...]uint16{
		0x25C6, // 0x60 `
		0x2592, // 0x61 a
		0x2409, // 0x62 b
		0x240C, // 0x63 c
		0x240D, // 0x64 d
		0x240A, // 0x65 e
		0x00B0, // 0x66 f
		0x00B1, // 0x67 g
		0x2424, // 0x68 h
		0x240B, // 0x69 i
		0x2518, // 0x6a j
		0x2510, // 0x6b k
		0x250C, // 0x6c l
		0x2514, // 0x6d m
		0x253C, // 0x6e n
		0x23BA, // 0x6f o
		0x23BB, // 0x70 p
		0x2500, // 0x71 q
		0x23BC, // 0x72 r
		0x23BD, // 0x73 s
		0x251C, // 0x74 t
		0x2524, // 0x75 u
		0x2534, // 0x76 v
		0x252C, // 0x77 w
		0x2502, // 0x78 x
		0x2264, // 0x79 y
		0x2265, // 0x7a z
		0x03C0, // 0x7b {
		0x2260, // 0x7c |
		0x00A3, // 0x7d }
		0x00B7, // 0x7e ~
	}
	expectEqual(t, 31, len(expected))

	for i, v := range tbl {
		if i >= 0x60 && i <= 0x7E {
			expectEqual(t, expected[i-0x60], v)
		} else {
			// 0x23 is NOT remapped here (only british does that); 0x5F,
			// 0x7F and everything above 0x7F is the identity.
			expectEqual(t, uint16(i), v)
		}
	}
}

// Zig: test "term51: table: the same charset returns the same storage and charsets differ"
func TestCharsets_term51_table_the_same_charset_returns_the_same_storage_and_charsets_differ(t *testing.T) {
	expectEqual(t, &CharsetsTable(CharsetsCharsetAscii)[0], &CharsetsTable(CharsetsCharsetAscii)[0])
	expectEqual(t, &CharsetsTable(CharsetsCharsetBritish)[0], &CharsetsTable(CharsetsCharsetBritish)[0])
	expectEqual(t, &CharsetsTable(CharsetsCharsetDecSpecial)[0], &CharsetsTable(CharsetsCharsetDecSpecial)[0])
	expect(t, &CharsetsTable(CharsetsCharsetAscii)[0] != &CharsetsTable(CharsetsCharsetBritish)[0], "table(.ascii).ptr != table(.british).ptr")
	expect(t, &CharsetsTable(CharsetsCharsetAscii)[0] != &CharsetsTable(CharsetsCharsetDecSpecial)[0], "table(.ascii).ptr != table(.dec_special).ptr")
	expect(t, &CharsetsTable(CharsetsCharsetBritish)[0] != &CharsetsTable(CharsetsCharsetDecSpecial)[0], "table(.british).ptr != table(.dec_special).ptr")
	// The british remap does not leak into the other tables.
	expectEqual(t, uint16(0x23), CharsetsTable(CharsetsCharsetAscii)[0x23])
	expectEqual(t, uint16(0xA3), CharsetsTable(CharsetsCharsetBritish)[0x23])
	expectEqual(t, uint16(0x23), CharsetsTable(CharsetsCharsetDecSpecial)[0x23])
}
