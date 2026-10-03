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
	expectEqual(t, ColorRGB{r: 255, g: 255, b: 255}, zigUnwrap(t, col_x11Get("white")))
	expectEqual(t, Some(ColorRGB{r: 0, g: 250, b: 154}), col_x11Get("medium spring green"))
	expectEqual(t, Some(ColorRGB{r: 34, g: 139, b: 34}), col_x11Get("ForestGreen"))
	expectEqual(t, Some(ColorRGB{r: 34, g: 139, b: 34}), col_x11Get("FoReStGReen"))
	expectEqual(t, Some(ColorRGB{r: 0, g: 0, b: 0}), col_x11Get("black"))
	expectEqual(t, Some(ColorRGB{r: 255, g: 0, b: 0}), col_x11Get("red"))
	expectEqual(t, Some(ColorRGB{r: 0, g: 255, b: 0}), col_x11Get("green"))
	expectEqual(t, Some(ColorRGB{r: 0, g: 0, b: 255}), col_x11Get("blue"))
	expectEqual(t, Some(ColorRGB{r: 255, g: 255, b: 255}), col_x11Get("white"))
	expectEqual(t, Some(ColorRGB{r: 124, g: 252, b: 0}), col_x11Get("lawngreen"))
	expectEqual(t, Some(ColorRGB{r: 0, g: 250, b: 154}), col_x11Get("mediumspringgreen"))
	expectEqual(t, Some(ColorRGB{r: 34, g: 139, b: 34}), col_x11Get("forestgreen"))
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
	expectEqual(t, ColorRGB{r: 255, g: 250, b: 250}, zigUnwrap(t, col_x11Get("snow")))
	expectEqual(t, ColorRGB{r: 0, g: 128, b: 128}, zigUnwrap(t, col_x11Get("teal")))
	expect(t, strings.HasPrefix(x11ColorData, "255 250 250\t\tsnow\n"), "std.mem.startsWith(u8, data, \"255 250 250\\t\\tsnow\\n\")")
	expect(t, strings.HasSuffix(x11ColorData, "  0 128 128\t\tteal\n"), "std.mem.endsWith(u8, data, \"  0 128 128\\t\\tteal\\n\")")
}

// Zig: test "term51: colorMap: spot-check name and value pairs across the table"
func TestX11Color_term51_colorMap_spot_check_name_and_value_pairs_across_the_table(t *testing.T) {
	cases := [...]struct {
		name  string
		color ColorRGB
	}{
		{"snow", ColorRGB{r: 255, g: 250, b: 250}},
		{"ghost white", ColorRGB{r: 248, g: 248, b: 255}},
		{"GhostWhite", ColorRGB{r: 248, g: 248, b: 255}},
		{"white smoke", ColorRGB{r: 245, g: 245, b: 245}},
		{"old lace", ColorRGB{r: 253, g: 245, b: 230}},
		{"antique white", ColorRGB{r: 250, g: 235, b: 215}},
		{"peach puff", ColorRGB{r: 255, g: 218, b: 185}},
		{"alice blue", ColorRGB{r: 240, g: 248, b: 255}},
		{"misty rose", ColorRGB{r: 255, g: 228, b: 225}},
		{"white", ColorRGB{r: 255, g: 255, b: 255}},
		{"black", ColorRGB{r: 0, g: 0, b: 0}},
		{"dark slate gray", ColorRGB{r: 47, g: 79, b: 79}},
		{"light slate gray", ColorRGB{r: 119, g: 136, b: 153}},
		{"LightSlateGrey", ColorRGB{r: 119, g: 136, b: 153}},
		{"gray", ColorRGB{r: 190, g: 190, b: 190}},
		{"grey", ColorRGB{r: 190, g: 190, b: 190}},
		{"x11 gray", ColorRGB{r: 190, g: 190, b: 190}},
		{"web gray", ColorRGB{r: 128, g: 128, b: 128}},
		{"midnight blue", ColorRGB{r: 25, g: 25, b: 112}},
		{"navy", ColorRGB{r: 0, g: 0, b: 128}},
		{"NavyBlue", ColorRGB{r: 0, g: 0, b: 128}},
		{"blue", ColorRGB{r: 0, g: 0, b: 255}},
		{"dodger blue", ColorRGB{r: 30, g: 144, b: 255}},
		{"deep sky blue", ColorRGB{r: 0, g: 191, b: 255}},
		{"cyan", ColorRGB{r: 0, g: 255, b: 255}},
		{"aqua", ColorRGB{r: 0, g: 255, b: 255}},
		{"green", ColorRGB{r: 0, g: 255, b: 0}},
		{"lime", ColorRGB{r: 0, g: 255, b: 0}},
		{"web green", ColorRGB{r: 0, g: 128, b: 0}},
		{"chartreuse", ColorRGB{r: 127, g: 255, b: 0}},
		{"yellow", ColorRGB{r: 255, g: 255, b: 0}},
		{"gold", ColorRGB{r: 255, g: 215, b: 0}},
		{"tan", ColorRGB{r: 210, g: 180, b: 140}},
		{"orange red", ColorRGB{r: 255, g: 69, b: 0}},
		{"red", ColorRGB{r: 255, g: 0, b: 0}},
		{"hot pink", ColorRGB{r: 255, g: 105, b: 180}},
		{"magenta", ColorRGB{r: 255, g: 0, b: 255}},
		{"fuchsia", ColorRGB{r: 255, g: 0, b: 255}},
		{"orchid", ColorRGB{r: 218, g: 112, b: 214}},
		{"maroon", ColorRGB{r: 176, g: 48, b: 96}},
		{"x11 maroon", ColorRGB{r: 176, g: 48, b: 96}},
		{"web maroon", ColorRGB{r: 128, g: 0, b: 0}},
		{"purple", ColorRGB{r: 160, g: 32, b: 240}},
		{"web purple", ColorRGB{r: 128, g: 0, b: 128}},
		{"tan1", ColorRGB{r: 255, g: 165, b: 79}},
		{"DeepSkyBlue4", ColorRGB{r: 0, g: 104, b: 139}},
		{"gray0", ColorRGB{r: 0, g: 0, b: 0}},
		{"gray1", ColorRGB{r: 3, g: 3, b: 3}},
		{"gray50", ColorRGB{r: 127, g: 127, b: 127}},
		{"grey51", ColorRGB{r: 130, g: 130, b: 130}},
		{"gray99", ColorRGB{r: 252, g: 252, b: 252}},
		{"gray100", ColorRGB{r: 255, g: 255, b: 255}},
		{"grey100", ColorRGB{r: 255, g: 255, b: 255}},
		{"dark gray", ColorRGB{r: 169, g: 169, b: 169}},
		{"DarkGray", ColorRGB{r: 169, g: 169, b: 169}},
		{"dark red", ColorRGB{r: 139, g: 0, b: 0}},
		{"light green", ColorRGB{r: 144, g: 238, b: 144}},
		{"crimson", ColorRGB{r: 220, g: 20, b: 60}},
		{"indigo", ColorRGB{r: 75, g: 0, b: 130}},
		{"olive", ColorRGB{r: 128, g: 128, b: 0}},
		{"rebecca purple", ColorRGB{r: 102, g: 51, b: 153}},
		{"RebeccaPurple", ColorRGB{r: 102, g: 51, b: 153}},
		{"silver", ColorRGB{r: 192, g: 192, b: 192}},
		{"teal", ColorRGB{r: 0, g: 128, b: 128}},
	}
	expect(t, len(cases) >= 20, "cases.len >= 20")
	for _, c := range cases {
		expectEqual(t, c.color, zigUnwrap(t, col_x11Get(c.name)))
	}
}

// Zig: test "term51: colorMap: lookup is case-insensitive and exact otherwise"
func TestX11Color_term51_colorMap_lookup_is_case_insensitive_and_exact_otherwise(t *testing.T) {
	forest := ColorRGB{r: 34, g: 139, b: 34}
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
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 128}, zigUnwrap(t, col_x11Get("navy")))
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 128}, zigUnwrap(t, col_x11Get("navy blue")))
	expectEqual(t, ColorRGB{r: 210, g: 180, b: 140}, zigUnwrap(t, col_x11Get("tan")))
	expectEqual(t, ColorRGB{r: 255, g: 165, b: 79}, zigUnwrap(t, col_x11Get("tan1")))
	expectEqual(t, ColorRGB{r: 190, g: 190, b: 190}, zigUnwrap(t, col_x11Get("gray")))
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 0}, zigUnwrap(t, col_x11Get("gray0")))
	expectEqual(t, ColorRGB{r: 238, g: 221, b: 130}, zigUnwrap(t, col_x11Get("light goldenrod")))
	expectEqual(t, ColorRGB{r: 250, g: 250, b: 210}, zigUnwrap(t, col_x11Get("light goldenrod yellow")))
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
		want := ColorRGB{r: uint8(r), g: uint8(g), b: uint8(b)}
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
	expectEqual(t, ColorRGB{r: 0, g: 250, b: 154}, zigUnwrap(t, col_x11Get("mediumspringgreen")))
	// ... but other spellings do not.
	expect(t, !col_x11Get("medium-spring-green").Valid, "map.get(\"medium-spring-green\") == null")
	expect(t, !col_x11Get("medium_spring_green").Valid, "map.get(\"medium_spring_green\") == null")
	expect(t, !col_x11Get("medium spring  green").Valid, "map.get(\"medium spring  green\") == null")
	expect(t, !col_x11Get("medium  spring green").Valid, "map.get(\"medium  spring green\") == null")
	expect(t, !col_x11Get(" medium spring green").Valid, "map.get(\" medium spring green\") == null")
	expect(t, !col_x11Get("medium spring green ").Valid, "map.get(\"medium spring green \") == null")
	expect(t, zigUnwrap(t, col_x11Get("medium spring green")).g == 250, "map.get(\"medium spring green\").?.g == 250")
	// an over-long name is rejected, not truncated
	long := strings.Repeat("a", 4096)
	expect(t, !col_x11Get(long).Valid, "map.get(long) == null")
}
