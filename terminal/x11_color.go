// WIREFRAME for src/terminal/x11_color.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.

package terminal

// X11ColorMap is x11_color.map: the map of all available X11 colors.
//
// Zig: `pub const map = colorMap()`, computed at compile time. Wireframe:
// left zero; the implementation initialises it once at start-up from
// colorMap().
var X11ColorMap X11ColorColorMap

// X11ColorColorMap is x11_color.ColorMap:
// std.StaticStringMapWithEql(RGB, eqlAsciiIgnoreCase). Lookup ignores ASCII
// case and is exact otherwise.
type X11ColorColorMap struct {
	kvs map[string]ColorRGB
}

// get is StaticStringMap.get: the value for name, or none.
func (self X11ColorColorMap) get(name []uint8) Optional[ColorRGB] { panic(notImplemented) }

// colorMap is x11_color.colorMap: builds the map by parsing the embedded
// rgb.txt, one entry per line.
func colorMap() X11ColorColorMap { panic(notImplemented) }

// x11ColorData is x11_color.data: the rgb.txt file from the X11 project.
// This was last sourced from this location:
// https://gitlab.freedesktop.org/xorg/app/rgb
// This data is licensed under the MIT/X11 license.
//
// Zig: @embedFile("res/rgb.txt"). Wireframe: empty. The implementation
// embeds upstream's file unchanged (terminal/res/rgb.txt via go:embed); the
// table is never retyped by hand.
var x11ColorData string
