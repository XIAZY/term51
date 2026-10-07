// Tests translated one for one from src/terminal/kitty/color.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualStrings and zigUnwrap are defined in Parser_test.go; helpers
// that do not exist there are prefixed col_.
//
// Not translated (listed in the report to the project lead):
//   - "term51: OSC.cval: always void, default terminator is st": a test of
//     the C binding helper (cval, C), which is not ported. It also checks
//     OSC.terminator, which the wireframe leaves commented out until osc.zig
//     is wired.
//
// Translation notes:
//   - `std.fmt.bufPrint(&buf, "{f}", .{k})` is k.Format() (docs/mapping.md).
//   - `Kind.parse(key).?.palette` and `.special` are zigUnwrap plus
//     col_kindPalette / col_kindSpecial, which check the union tag as
//     Zig's safety-checked access does. Comparing two Kinds is
//     col_expectKindEqual (only the payload of the active tag counts).
//   - `inline for (@typeInfo(Special).@"enum".fields)` loops over
//     KittyColorSpecial(0) .. KittyColorSpecial(kittyColorSpecialCount-1),
//     the values in declaration order; `f.name` is the String() of the value.
//   - `null` compared with an optional is None[KittyColorKind]().

package terminal

import (
	"testing"
)

// col_kindPalette is the safety-checked union access `k.palette`.
func col_kindPalette(t *testing.T, k KittyColorKind) uint8 {
	t.Helper()
	if k.Tag != KittyColorKindPalette {
		t.Fatalf("access of union field palette while field %v is active", k.Tag)
	}
	return k.Palette
}

// col_kindSpecial is the safety-checked union access `k.special`.
func col_kindSpecial(t *testing.T, k KittyColorKind) KittyColorSpecial {
	t.Helper()
	if k.Tag != KittyColorKindSpecial {
		t.Fatalf("access of union field special while field %v is active", k.Tag)
	}
	return k.Special
}

// col_expectKindEqual is Zig's `try testing.expectEqual(expected, actual)` on
// two Kind unions: the tags and the payloads of the active tag are compared.
func col_expectKindEqual(t *testing.T, want KittyColorKind, got KittyColorKind) {
	t.Helper()
	if want.Tag != got.Tag {
		t.Fatalf("got tag %v, want tag %v", got.Tag, want.Tag)
	}
	switch want.Tag {
	case KittyColorKindPalette:
		expectEqual(t, want.Palette, got.Palette)
	case KittyColorKindSpecial:
		expectEqual(t, want.Special, got.Special)
	}
}

// Zig: test "OSC: kitty color protocol kind string"
func TestKittyColor_OSC_kitty_color_protocol_kind_string(t *testing.T) {
	// Zig-only: var buf: [256]u8 and std.fmt.bufPrint(&buf, "{f}", ...);
	// Go: Format() returns the string.
	{
		actual := KittyColorKind{Tag: KittyColorKindSpecial, Special: KittyColorSpecialForeground}.Format()
		expectEqualStrings(t, "foreground", actual)
	}
	{
		actual := KittyColorKind{Tag: KittyColorKindPalette, Palette: 42}.Format()
		expectEqualStrings(t, "42", actual)
	}
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// term51Fmt is the term51 test helper: format a Kind. Zig formats into the
// given buffer and can fail with NoSpaceLeft; Format() returns the string.
func term51Fmt(k KittyColorKind) string {
	return k.Format()
}

var term51_special_names = [...]string{
	"foreground",
	"background",
	"selection_foreground",
	"selection_background",
	"cursor",
	"cursor_text",
	"visual_bell",
	"second_transparent_background",
}

// Zig: test "term51: OSC.deinit: frees the request list"
func TestKittyColor_term51_OSC_deinit_frees_the_request_list(t *testing.T) {
	// Zig-only: const alloc = testing.allocator; the allocator argument is
	// dropped, and the testing allocator's leak report has no Go
	// counterpart.
	osc := KittyColorOSC{}
	osc.List = append(osc.List, KittyColorOSCRequest{Tag: KittyColorOSCRequestQuery, Query: KittyColorKind{Tag: KittyColorKindPalette, Palette: 1}})
	osc.List = append(osc.List, KittyColorOSCRequest{Tag: KittyColorOSCRequestSet, Set: KittyColorOSCRequestSetPayload{
		Key:   KittyColorKind{Tag: KittyColorKindSpecial, Special: KittyColorSpecialCursor},
		Color: ColorRGB{R: 1, G: 2, B: 3},
	}})
	osc.List = append(osc.List, KittyColorOSCRequest{Tag: KittyColorOSCRequestReset, Reset: KittyColorKind{Tag: KittyColorKindPalette, Palette: 255}})
	expectEqual(t, 3, len(osc.List))
	// The testing allocator reports a leak if deinit does not free.
	osc.Deinit()

	// Empty list (never allocated) is also fine.
	empty := KittyColorOSC{}
	empty.Deinit()
}

// Zig: test "term51: Kind.parse: every special key name"
func TestKittyColor_term51_Kind_parse_every_special_key_name(t *testing.T) {
	expectEqual(t, 8, len(term51_special_names))
	expectEqual(t, 8, kittyColorSpecialCount) // @typeInfo(Special).@"enum".fields.len
	for i := 0; i < kittyColorSpecialCount; i++ {
		f := KittyColorSpecial(i)
		expectEqualStrings(t, f.String(), term51_special_names[i])
		k := zigUnwrap(t, KittyColorKindParse([]uint8(f.String())))
		expectEqual(t, f, col_kindSpecial(t, k))
	}
	expectEqual(t, KittyColorSpecialForeground, col_kindSpecial(t, zigUnwrap(t, KittyColorKindParse([]uint8("foreground")))))
	expectEqual(t, KittyColorSpecialSecondTransparentBackground, col_kindSpecial(t, zigUnwrap(t, KittyColorKindParse([]uint8("second_transparent_background")))))
}

// Zig: test "term51: Kind.parse: palette indexes and boundaries"
func TestKittyColor_term51_Kind_parse_palette_indexes_and_boundaries(t *testing.T) {
	expectEqual(t, uint8(0), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("0")))))
	expectEqual(t, uint8(1), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("1")))))
	expectEqual(t, uint8(42), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("42")))))
	expectEqual(t, uint8(255), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("255")))))
	// One past the limit.
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("256")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("1000")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("99999999999999999999")))
	// Leading zeros are decimal.
	expectEqual(t, uint8(7), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("007")))))
	expectEqual(t, uint8(0), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("000")))))
	expectEqual(t, uint8(255), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("0255")))))
	// Underscores are accepted by std.fmt.parseUnsigned between digits.
	expectEqual(t, uint8(10), col_kindPalette(t, zigUnwrap(t, KittyColorKindParse([]uint8("1_0")))))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("_1")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("1_")))
}

// Zig: test "term51: Kind.parse: invalid, empty, wrong case, and whitespace"
func TestKittyColor_term51_Kind_parse_invalid_empty_wrong_case_and_whitespace(t *testing.T) {
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("-1")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("-0")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("+1")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("0x10")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("1.5")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("abc")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8(" 1")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("1 ")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8(" foreground")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("foreground ")))
	// Special names are case sensitive.
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("Foreground")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("FOREGROUND")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("Cursor_Text")))
	// Names that are not specials, prefixes, and enum-like spellings.
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("fore")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("cursor-text")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("selection")))
	expectEqual(t, None[KittyColorKind](), KittyColorKindParse([]uint8("palette")))
}

// Zig: test "term51: Kind.format: every special and palette boundary"
func TestKittyColor_term51_Kind_format_every_special_and_palette_boundary(t *testing.T) {
	// Zig-only: var buf: [64]u8; Format() needs no buffer.
	for i := 0; i < kittyColorSpecialCount; i++ {
		f := KittyColorSpecial(i)
		s := term51Fmt(KittyColorKind{Tag: KittyColorKindSpecial, Special: f})
		expectEqualStrings(t, f.String(), s)
	}
	expectEqualStrings(t, "0", term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 0}))
	expectEqualStrings(t, "9", term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 9}))
	expectEqualStrings(t, "10", term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 10}))
	expectEqualStrings(t, "100", term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 100}))
	expectEqualStrings(t, "255", term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 255}))
}

// Zig: test "term51: Kind.format: round trip through parse for every kind"
func TestKittyColor_term51_Kind_format_round_trip_through_parse_for_every_kind(t *testing.T) {
	// Zig-only: var buf: [64]u8; Format() needs no buffer.
	for i := 0; i <= 255; i++ {
		k := KittyColorKind{Tag: KittyColorKindPalette, Palette: uint8(i)}
		k2 := zigUnwrap(t, KittyColorKindParse([]uint8(term51Fmt(k))))
		col_expectKindEqual(t, k, k2)
	}
	for i := 0; i < kittyColorSpecialCount; i++ {
		k := KittyColorKind{Tag: KittyColorKindSpecial, Special: KittyColorSpecial(i)}
		k2 := zigUnwrap(t, KittyColorKindParse([]uint8(term51Fmt(k))))
		col_expectKindEqual(t, k, k2)
	}
}

// Zig: test "term51: Kind.format: too small a buffer is an error"
func TestKittyColor_term51_Kind_format_too_small_a_buffer_is_an_error(t *testing.T) {
	// Zig-only: var buf: [2]u8 and
	// testing.expectError(error.NoSpaceLeft, term51Fmt(&buf, ...)). Format()
	// returns a string and cannot run out of space; the nearest Go check is
	// that the text does not fit in the 2 byte buffer.
	expect(t, len(term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 255})) > 2, "\"255\" does not fit in [2]u8")
	expect(t, len(term51Fmt(KittyColorKind{Tag: KittyColorKindSpecial, Special: KittyColorSpecialCursor})) > 2, "\"cursor\" does not fit in [2]u8")
	// Zig-only: var buf3: [3]u8; the exact fit is 3 bytes of text.
	expectEqualStrings(t, "255", term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 255}))
	expectEqual(t, 3, len(term51Fmt(KittyColorKind{Tag: KittyColorKindPalette, Palette: 255})))
}

// Zig: test "term51: Kind.max: palette size plus special count"
func TestKittyColor_term51_Kind_max_palette_size_plus_special_count(t *testing.T) {
	expectEqual(t, 263, KittyColorKindMax)
	expectEqual(t, 255+8, KittyColorKindMax)
	// Zig-only: try testing.expect(@TypeOf(Kind.max) == usize);
	// Go: the assignment below is checked by the compiler.
	var max int = KittyColorKindMax
	_ = max
}
