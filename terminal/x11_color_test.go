// Tests translated one for one from src/terminal/x11_color.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualStrings and zigUnwrap are defined in Parser_test.go; helpers
// that do not exist there are prefixed col_ (the ones for color.zig are in
// color_test.go).
//
// Translation notes:
//   - `map` is X11ColorMap and `data` is x11ColorData.
//   - `map.get(name)` returns an Optional: a Zig comparison of an RGB with
//     the optional is Some(rgb), `.?` is zigUnwrap, and `null` is
//     None[ColorRGB]().
//   - `map.keys()` and `map.values()` are the keys and values of the Go map
//     X11ColorMap.kvs (a Go map has no order; the tests that use them do not
//     depend on it).
//   - Allocator arguments and frees have no Go counterpart.

package terminal

import (
	"strconv"
	"strings"
	"testing"
)

// col_eqlAsciiIgnoreCase is std.ascii.eqlIgnoreCase.
func col_eqlAsciiIgnoreCase(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if col_asciiLower(a[i]) != col_asciiLower(b[i]) {
			return false
		}
	}
	return true
}

func col_asciiLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func col_asciiUpper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}

// col_allocLowerString is std.ascii.allocLowerString.
func col_allocLowerString(s string) string {
	b := []byte(s)
	for i := range b {
		b[i] = col_asciiLower(b[i])
	}
	return string(b)
}

// col_allocUpperString is std.ascii.allocUpperString.
func col_allocUpperString(s string) string {
	b := []byte(s)
	for i := range b {
		b[i] = col_asciiUpper(b[i])
	}
	return string(b)
}

// col_x11Keys is map.keys(): the names of X11ColorMap.
func col_x11Keys() []string {
	keys := make([]string, 0, len(X11ColorMap.kvs))
	for k := range X11ColorMap.kvs {
		keys = append(keys, k)
	}
	return keys
}

// col_x11Get is map.get(name).
func col_x11Get(name string) Optional[ColorRGB] {
	return X11ColorMap.get([]uint8(name))
}

// Zig: test (unnamed, x11_color.zig:48)
func TestX11Color_unnamed(t *testing.T) {
	expectEqual(t, None[ColorRGB](), col_x11Get("nosuchcolor"))
	expectEqual(t, ColorRGB{R: 255, G: 255, B: 255}, zigUnwrap(t, col_x11Get("white")))
	expectEqual(t, Some(ColorRGB{R: 0, G: 250, B: 154}), col_x11Get("medium spring green"))
	expectEqual(t, Some(ColorRGB{R: 34, G: 139, B: 34}), col_x11Get("ForestGreen"))
	expectEqual(t, Some(ColorRGB{R: 34, G: 139, B: 34}), col_x11Get("FoReStGReen"))
	expectEqual(t, Some(ColorRGB{R: 0, G: 0, B: 0}), col_x11Get("black"))
	expectEqual(t, Some(ColorRGB{R: 255, G: 0, B: 0}), col_x11Get("red"))
	expectEqual(t, Some(ColorRGB{R: 0, G: 255, B: 0}), col_x11Get("green"))
	expectEqual(t, Some(ColorRGB{R: 0, G: 0, B: 255}), col_x11Get("blue"))
	expectEqual(t, Some(ColorRGB{R: 255, G: 255, B: 255}), col_x11Get("white"))
	expectEqual(t, Some(ColorRGB{R: 124, G: 252, B: 0}), col_x11Get("lawngreen"))
	expectEqual(t, Some(ColorRGB{R: 0, G: 250, B: 154}), col_x11Get("mediumspringgreen"))
	expectEqual(t, Some(ColorRGB{R: 34, G: 139, B: 34}), col_x11Get("forestgreen"))
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: colorMap: table has 782 entries, first is snow and last is teal"
func TestX11Color_term51_colorMap_table_has_782_entries_first_is_snow_and_last_is_teal(t *testing.T) {
	expectEqual(t, 782, len(X11ColorMap.kvs)) // map.keys().len
	expectEqual(t, 782, len(X11ColorMap.kvs)) // map.values().len
	expectEqual(t, 782, strings.Count(x11ColorData, "\n"))

	// Every key is non-empty, trimmed, and distinct ignoring ASCII case.
	keys := col_x11Keys()
	for i, k := range keys {
		expect(t, len(k) > 0, "k.len > 0")
		expectEqualStrings(t, strings.Trim(k, " \t\n"), k)
		for _, k2 := range keys[i+1:] {
			expect(t, !col_eqlAsciiIgnoreCase(k, k2), "!std.ascii.eqlIgnoreCase(k, k2)")
		}
	}

	// First and last lines of rgb.txt.
	expectEqual(t, ColorRGB{R: 255, G: 250, B: 250}, zigUnwrap(t, col_x11Get("snow")))
	expectEqual(t, ColorRGB{R: 0, G: 128, B: 128}, zigUnwrap(t, col_x11Get("teal")))
	expect(t, strings.HasPrefix(x11ColorData, "255 250 250\t\tsnow\n"), "std.mem.startsWith(u8, data, \"255 250 250\\t\\tsnow\\n\")")
	expect(t, strings.HasSuffix(x11ColorData, "  0 128 128\t\tteal\n"), "std.mem.endsWith(u8, data, \"  0 128 128\\t\\tteal\\n\")")
}

// Zig: test "term51: colorMap: spot-check name and value pairs across the table"
func TestX11Color_term51_colorMap_spot_check_name_and_value_pairs_across_the_table(t *testing.T) {
	cases := [...]struct {
		name  string
		color ColorRGB
	}{
		{"snow", ColorRGB{R: 255, G: 250, B: 250}},
		{"ghost white", ColorRGB{R: 248, G: 248, B: 255}},
		{"GhostWhite", ColorRGB{R: 248, G: 248, B: 255}},
		{"white smoke", ColorRGB{R: 245, G: 245, B: 245}},
		{"old lace", ColorRGB{R: 253, G: 245, B: 230}},
		{"antique white", ColorRGB{R: 250, G: 235, B: 215}},
		{"peach puff", ColorRGB{R: 255, G: 218, B: 185}},
		{"alice blue", ColorRGB{R: 240, G: 248, B: 255}},
		{"misty rose", ColorRGB{R: 255, G: 228, B: 225}},
		{"white", ColorRGB{R: 255, G: 255, B: 255}},
		{"black", ColorRGB{R: 0, G: 0, B: 0}},
		{"dark slate gray", ColorRGB{R: 47, G: 79, B: 79}},
		{"light slate gray", ColorRGB{R: 119, G: 136, B: 153}},
		{"LightSlateGrey", ColorRGB{R: 119, G: 136, B: 153}},
		{"gray", ColorRGB{R: 190, G: 190, B: 190}},
		{"grey", ColorRGB{R: 190, G: 190, B: 190}},
		{"x11 gray", ColorRGB{R: 190, G: 190, B: 190}},
		{"web gray", ColorRGB{R: 128, G: 128, B: 128}},
		{"midnight blue", ColorRGB{R: 25, G: 25, B: 112}},
		{"navy", ColorRGB{R: 0, G: 0, B: 128}},
		{"NavyBlue", ColorRGB{R: 0, G: 0, B: 128}},
		{"blue", ColorRGB{R: 0, G: 0, B: 255}},
		{"dodger blue", ColorRGB{R: 30, G: 144, B: 255}},
		{"deep sky blue", ColorRGB{R: 0, G: 191, B: 255}},
		{"cyan", ColorRGB{R: 0, G: 255, B: 255}},
		{"aqua", ColorRGB{R: 0, G: 255, B: 255}},
		{"green", ColorRGB{R: 0, G: 255, B: 0}},
		{"lime", ColorRGB{R: 0, G: 255, B: 0}},
		{"web green", ColorRGB{R: 0, G: 128, B: 0}},
		{"chartreuse", ColorRGB{R: 127, G: 255, B: 0}},
		{"yellow", ColorRGB{R: 255, G: 255, B: 0}},
		{"gold", ColorRGB{R: 255, G: 215, B: 0}},
		{"tan", ColorRGB{R: 210, G: 180, B: 140}},
		{"orange red", ColorRGB{R: 255, G: 69, B: 0}},
		{"red", ColorRGB{R: 255, G: 0, B: 0}},
		{"hot pink", ColorRGB{R: 255, G: 105, B: 180}},
		{"magenta", ColorRGB{R: 255, G: 0, B: 255}},
		{"fuchsia", ColorRGB{R: 255, G: 0, B: 255}},
		{"orchid", ColorRGB{R: 218, G: 112, B: 214}},
		{"maroon", ColorRGB{R: 176, G: 48, B: 96}},
		{"x11 maroon", ColorRGB{R: 176, G: 48, B: 96}},
		{"web maroon", ColorRGB{R: 128, G: 0, B: 0}},
		{"purple", ColorRGB{R: 160, G: 32, B: 240}},
		{"web purple", ColorRGB{R: 128, G: 0, B: 128}},
		{"tan1", ColorRGB{R: 255, G: 165, B: 79}},
		{"DeepSkyBlue4", ColorRGB{R: 0, G: 104, B: 139}},
		{"gray0", ColorRGB{R: 0, G: 0, B: 0}},
		{"gray1", ColorRGB{R: 3, G: 3, B: 3}},
		{"gray50", ColorRGB{R: 127, G: 127, B: 127}},
		{"grey51", ColorRGB{R: 130, G: 130, B: 130}},
		{"gray99", ColorRGB{R: 252, G: 252, B: 252}},
		{"gray100", ColorRGB{R: 255, G: 255, B: 255}},
		{"grey100", ColorRGB{R: 255, G: 255, B: 255}},
		{"dark gray", ColorRGB{R: 169, G: 169, B: 169}},
		{"DarkGray", ColorRGB{R: 169, G: 169, B: 169}},
		{"dark red", ColorRGB{R: 139, G: 0, B: 0}},
		{"light green", ColorRGB{R: 144, G: 238, B: 144}},
		{"crimson", ColorRGB{R: 220, G: 20, B: 60}},
		{"indigo", ColorRGB{R: 75, G: 0, B: 130}},
		{"olive", ColorRGB{R: 128, G: 128, B: 0}},
		{"rebecca purple", ColorRGB{R: 102, G: 51, B: 153}},
		{"RebeccaPurple", ColorRGB{R: 102, G: 51, B: 153}},
		{"silver", ColorRGB{R: 192, G: 192, B: 192}},
		{"teal", ColorRGB{R: 0, G: 128, B: 128}},
	}
	expect(t, len(cases) >= 20, "cases.len >= 20")
	for _, c := range cases {
		expectEqual(t, c.color, zigUnwrap(t, col_x11Get(c.name)))
	}
}

// Zig: test "term51: colorMap: lookup is case-insensitive and exact otherwise"
func TestX11Color_term51_colorMap_lookup_is_case_insensitive_and_exact_otherwise(t *testing.T) {
	forest := ColorRGB{R: 34, G: 139, B: 34}
	expectEqual(t, forest, zigUnwrap(t, col_x11Get("forestgreen")))
	expectEqual(t, forest, zigUnwrap(t, col_x11Get("ForestGreen")))
	expectEqual(t, forest, zigUnwrap(t, col_x11Get("FORESTGREEN")))
	expectEqual(t, forest, zigUnwrap(t, col_x11Get("forest green")))
	expectEqual(t, forest, zigUnwrap(t, col_x11Get("FOREST GREEN")))
	expectEqual(t, forest, zigUnwrap(t, col_x11Get("Forest Green")))
	// Names with and without spaces are distinct keys.
	expectEqual(t, None[ColorRGB](), col_x11Get("forest  green"))
	expectEqual(t, None[ColorRGB](), col_x11Get("forest-green"))
	expectEqual(t, None[ColorRGB](), col_x11Get("forest_green"))
}

// Zig: test "term51: colorMap: names that are prefixes of other names"
func TestX11Color_term51_colorMap_names_that_are_prefixes_of_other_names(t *testing.T) {
	// "navy" vs "navy blue", "tan" vs "tan1", "gray" vs "gray0"/"gray100".
	expectEqual(t, ColorRGB{R: 0, G: 0, B: 128}, zigUnwrap(t, col_x11Get("navy")))
	expectEqual(t, ColorRGB{R: 0, G: 0, B: 128}, zigUnwrap(t, col_x11Get("navy blue")))
	expectEqual(t, ColorRGB{R: 210, G: 180, B: 140}, zigUnwrap(t, col_x11Get("tan")))
	expectEqual(t, ColorRGB{R: 255, G: 165, B: 79}, zigUnwrap(t, col_x11Get("tan1")))
	expectEqual(t, ColorRGB{R: 190, G: 190, B: 190}, zigUnwrap(t, col_x11Get("gray")))
	expectEqual(t, ColorRGB{R: 0, G: 0, B: 0}, zigUnwrap(t, col_x11Get("gray0")))
	expectEqual(t, ColorRGB{R: 238, G: 221, B: 130}, zigUnwrap(t, col_x11Get("light goldenrod")))
	expectEqual(t, ColorRGB{R: 250, G: 250, B: 210}, zigUnwrap(t, col_x11Get("light goldenrod yellow")))
	// A strict prefix of a name that is not itself a name is not found.
	expectEqual(t, None[ColorRGB](), col_x11Get("light golden"))
	expectEqual(t, None[ColorRGB](), col_x11Get("forestgree"))
	expectEqual(t, None[ColorRGB](), col_x11Get("whit"))
	expectEqual(t, None[ColorRGB](), col_x11Get("gray1000"))
	expectEqual(t, None[ColorRGB](), col_x11Get("gray101"))
	expectEqual(t, None[ColorRGB](), col_x11Get("navy blu"))
	// A name with extra suffix is not found.
	expectEqual(t, None[ColorRGB](), col_x11Get("whitee"))
	expectEqual(t, None[ColorRGB](), col_x11Get("red "))
	expectEqual(t, None[ColorRGB](), col_x11Get(" red"))
}

// Zig: test "term51: colorMap: unknown, empty, and unusual names"
func TestX11Color_term51_colorMap_unknown_empty_and_unusual_names(t *testing.T) {
	expectEqual(t, None[ColorRGB](), col_x11Get(""))
	expectEqual(t, None[ColorRGB](), col_x11Get(" "))
	expectEqual(t, None[ColorRGB](), col_x11Get("nosuchcolor"))
	expectEqual(t, None[ColorRGB](), col_x11Get("rgb"))
	expectEqual(t, None[ColorRGB](), col_x11Get("#ffffff"))
	expectEqual(t, None[ColorRGB](), col_x11Get("red\x00"))
	expectEqual(t, None[ColorRGB](), col_x11Get("\tred"))
	expectEqual(t, None[ColorRGB](), col_x11Get("gray101"))
	// Non-ASCII bytes never match.
	expectEqual(t, None[ColorRGB](), col_x11Get("r\xc3\xa9d"))
}

// Zig: test "term51: colorMap: every name in the data file resolves to its recorded value"
func TestX11Color_term51_colorMap_every_name_in_the_data_file_resolves_to_its_recorded_value(t *testing.T) {
	// Re-reads res/rgb.txt line by line, independently of colorMap(), and
	// checks the lookup for each name as written, upper-cased and
	// lower-cased. Duplicate names (case-insensitively) would make a later
	// line win, so first check there are none.
	// Zig-only: the std.StringHashMap with allocator-owned keys and its
	// deferred cleanup; Go uses a map[string]struct{} and the garbage
	// collector.
	seen := map[string]struct{}{}
	count := 0
	for _, line := range strings.Split(x11ColorData, "\n") {
		if len(line) == 0 {
			continue
		}
		r, err := strconv.ParseUint(strings.Trim(line[0:3], " "), 10, 8)
		zigTry(t, err)
		g, err := strconv.ParseUint(strings.Trim(line[4:7], " "), 10, 8)
		zigTry(t, err)
		b, err := strconv.ParseUint(strings.Trim(line[8:11], " "), 10, 8)
		zigTry(t, err)
		name := strings.Trim(line[12:], " \t\n")
		want := ColorRGB{R: uint8(r), G: uint8(g), B: uint8(b)}
		expect(t, len(name) > 0, "name.len > 0")

		lower := col_allocLowerString(name)
		upper := col_allocUpperString(name)

		expectEqual(t, want, zigUnwrap(t, col_x11Get(name)))
		expectEqual(t, want, zigUnwrap(t, col_x11Get(lower)))
		expectEqual(t, want, zigUnwrap(t, col_x11Get(upper)))

		_, found_existing := seen[lower]
		seen[lower] = struct{}{}
		expect(t, !found_existing, "!gop.found_existing")
		count += 1
	}
	expectEqual(t, 782, count)
}

// Zig: test "term51: colorMap: names with a changed or extra space do not resolve"
func TestX11Color_term51_colorMap_names_with_a_changed_or_extra_space_do_not_resolve(t *testing.T) {
	// The data file lists both "medium spring green" and "MediumSpringGreen",
	// so the run-together spelling resolves (case-insensitively) ...
	expectEqual(t, ColorRGB{R: 0, G: 250, B: 154}, zigUnwrap(t, col_x11Get("mediumspringgreen")))
	// ... but other spellings do not.
	expect(t, !col_x11Get("medium-spring-green").Valid, "map.get(\"medium-spring-green\") == null")
	expect(t, !col_x11Get("medium_spring_green").Valid, "map.get(\"medium_spring_green\") == null")
	expect(t, !col_x11Get("medium spring  green").Valid, "map.get(\"medium spring  green\") == null")
	expect(t, !col_x11Get("medium  spring green").Valid, "map.get(\"medium  spring green\") == null")
	expect(t, !col_x11Get(" medium spring green").Valid, "map.get(\" medium spring green\") == null")
	expect(t, !col_x11Get("medium spring green ").Valid, "map.get(\"medium spring green \") == null")
	expect(t, zigUnwrap(t, col_x11Get("medium spring green")).G == 250, "map.get(\"medium spring green\").?.g == 250")
	// an over-long name is rejected, not truncated
	long := strings.Repeat("a", 4096)
	expect(t, !col_x11Get(long).Valid, "map.get(long) == null")
}
