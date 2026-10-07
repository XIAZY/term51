// Command icons draws the menu's three icons that the system has none for
// (new tab, tabs, paste) into assets/images: white line drawings on a clear
// background, 81 by 81 pixels, the size BlackBerry 10 uses for action icons.
// Run from the repository's root: go run ./tools/icons
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

const size, line = 81, 5

type icon struct{ *image.NRGBA }

// box fills the rectangle x0, y0 to x1, y1 (the end not included).
func (i icon) box(x0, y0, x1, y1 int, c color.NRGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i.SetNRGBA(x, y, c)
		}
	}
}

// frame draws the outline of a rectangle, the line inside it.
func (i icon) frame(x0, y0, x1, y1 int) {
	i.box(x0, y0, x1, y0+line, white)
	i.box(x0, y1-line, x1, y1, white)
	i.box(x0, y0, x0+line, y1, white)
	i.box(x1-line, y0, x1, y1, white)
}

var white, clear = color.NRGBA{255, 255, 255, 255}, color.NRGBA{}

func main() {
	draw := map[string]func(icon){
		// A plus.
		"newtab": func(i icon) {
			i.box(38, 16, 38+line, 65, white)
			i.box(16, 38, 65, 38+line, white)
		},
		// Two sheets, one in front of the other.
		"tabs": func(i icon) {
			i.frame(28, 14, 67, 53)
			i.box(14, 28, 53, 67, clear)
			i.frame(14, 28, 53, 67)
		},
		// A clipboard: the board and its clip.
		"paste": func(i icon) {
			i.frame(18, 20, 63, 69)
			i.box(30, 12, 51, 26, white)
		},
	}
	for name, f := range draw {
		i := icon{image.NewNRGBA(image.Rect(0, 0, size, size))}
		f(i)
		out, err := os.Create("assets/images/" + name + ".png")
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(out, i); err != nil {
			log.Fatal(err)
		}
		if err := out.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
