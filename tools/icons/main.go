// Command icons draws the menus' icons that the system has none for (new
// tab, close tab, tabs, paste, copy, select all, cancel, settings, the three
// dots) into assets/images: white line drawings on a clear
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

// sheets draws two sheets, one in front of the other.
func sheets(i icon) {
	i.frame(28, 14, 67, 53)
	i.box(14, 28, 53, 67, clear)
	i.frame(14, 28, 53, 67)
}

func main() {
	draw := map[string]func(icon){
		// A plus.
		"newtab": func(i icon) {
			i.box(38, 16, 38+line, 65, white)
			i.box(16, 38, 65, 38+line, white)
		},
		// A cross.
		"closetab": func(i icon) {
			for y := 18; y < 63; y++ {
				for x := 18; x < 63; x++ {
					if d := x - y; d >= -line/2-1 && d <= line/2+1 {
						i.SetNRGBA(x, y, white)
						i.SetNRGBA(size-1-x, y, white)
					}
				}
			}
		},
		// Three dots, one above the other.
		"dots": func(i icon) {
			for _, y := range []int{16, 37, 58} {
				i.box(36, y, 45, y+9, white)
			}
		},
		// Three sliders: lines with a knob each.
		"settings": func(i icon) {
			for n, knob := range []int{24, 48, 32} {
				y := 20 + 20*n
				i.box(14, y, 67, y+line, white)
				i.box(knob, y-6, knob+line+4, y+line+6, white)
			}
		},
		// Two sheets, one in front of the other.
		"tabs": sheets,
		"copy": sheets,
		// A dashed frame around a filled square.
		"selectall": func(i icon) {
			i.frame(14, 14, 67, 67)
			for _, at := range []int{26, 50} {
				i.box(at, 14, at+line, 67, clear)
				i.box(14, at, 67, at+line, clear)
			}
			i.box(31, 31, 50, 50, white)
		},
		// A ring with a stroke through it.
		"cancel": func(i icon) {
			const c, outer, inner = 40, 27, 27 - line
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					d := (x-c)*(x-c) + (y-c)*(y-c)
					ring := d <= outer*outer && d >= inner*inner
					stroke := d <= outer*outer && x+y >= 2*c-line/2-1 && x+y <= 2*c+line/2+1
					if ring || stroke {
						i.SetNRGBA(x, y, white)
					}
				}
			}
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
