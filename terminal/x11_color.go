// Port of src/terminal/x11_color.zig (Ghostty v1.3.1).

package terminal

import (
	_ "embed"
	"strings"
)

// X11ColorMap is x11_color.map: the map of all available X11 colors.
//
// Zig: `pub const map = colorMap()`, computed at compile time. Go computes it
// once at start-up.
var X11ColorMap X11ColorColorMap = colorMap()

// X11ColorColorMap is x11_color.ColorMap:
// std.StaticStringMapWithEql(RGB, eqlAsciiIgnoreCase). Lookup ignores ASCII
// case and is exact otherwise.
//
// Go: a map keyed by the name with its ASCII letters lower-cased
// (x11ColorAsciiLower); get lower-cases the name in the same way, which is
// the comparison eqlAsciiIgnoreCase makes. The names of rgb.txt are all
// distinct ignoring ASCII case (782 entries, 782 keys), so no entry hides
// another.
type X11ColorColorMap struct {
	kvs map[string]ColorRGB
}

// get is StaticStringMap.get: the value for name, or none.
func (self X11ColorColorMap) get(name []uint8) Optional[ColorRGB] {
	if v, ok := self.kvs[x11ColorAsciiLower(name)]; ok {
		return Some(v)
	}
	return None[ColorRGB]()
}

// x11ColorAsciiLower is the key of name in X11ColorColorMap.kvs: name with
// 'A' to 'Z' replaced by 'a' to 'z' (std.ascii.toLower on each byte; other
// bytes, including non-ASCII ones, are unchanged).
func x11ColorAsciiLower(name []uint8) string {
	b := make([]byte, len(name))
	for i, c := range name {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// colorMap is x11_color.colorMap: builds the map by parsing the embedded
// rgb.txt, one entry per line.
//
// Zig runs it at compile time, where a parse error is a compile error; Go
// runs it at start-up and panics instead.
func colorMap() X11ColorColorMap {
	// The length of our data is the number of lines in the rgb file.
	len_ := strings.Count(x11ColorData, "\n")
	kvs := make(map[string]ColorRGB, len_)

	// Parse the line. This is not very robust parsing, because we expect
	// a very exact format for rgb.txt. However, this is all done at comptime
	// so if our data is bad, we should hopefully get an error here or one
	// of our unit tests will catch it.
	i := 0
	for _, line := range strings.Split(x11ColorData, "\n") {
		if len(line) == 0 {
			continue
		}
		r := x11ColorParseIntU8(colorStdMemTrim([]uint8(line[0:3]), " "))
		g := x11ColorParseIntU8(colorStdMemTrim([]uint8(line[4:7]), " "))
		b := x11ColorParseIntU8(colorStdMemTrim([]uint8(line[8:11]), " "))
		name := colorStdMemTrim([]uint8(line[12:]), " \t\n")
		kvs[x11ColorAsciiLower(name)] = ColorRGB{r: r, g: g, b: b}
		i += 1
	}
	assert(i == len_)

	return X11ColorColorMap{kvs: kvs}
}

// x11ColorParseIntU8 is `try std.fmt.parseInt(u8, buf, 10)` in colorMap:
// an optional '+' or '-' (only "-0" fits a u8) and then parseUnsigned's
// rules. Not a Zig function. A parse error, a compile error in Zig, panics.
func x11ColorParseIntU8(buf []uint8) uint8 {
	if len(buf) == 0 {
		panic("x11_color: InvalidCharacter")
	}
	negative := false
	if buf[0] == '+' || buf[0] == '-' {
		negative = buf[0] == '-'
		buf = buf[1:]
	}
	v, ok := colorStdFmtParseUnsigned(buf, 10, 8)
	if !ok || (negative && v != 0) {
		panic("x11_color: invalid number in rgb.txt")
	}
	return uint8(v)
}

// x11ColorData is x11_color.data: the rgb.txt file from the X11 project.
// This was last sourced from this location:
// https://gitlab.freedesktop.org/xorg/app/rgb
// This data is licensed under the MIT/X11 license.
//
// Zig: @embedFile("res/rgb.txt"). Go embeds a byte-for-byte copy of
// upstream's file, terminal/res/rgb.txt; the table is never retyped by hand
// (TestX11Color_corpus_rgb_txt compares the copy with the pinned upstream
// file).
//
//go:embed res/rgb.txt
var x11ColorData string
