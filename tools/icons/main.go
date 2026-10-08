// Command icons draws the icons the system has none for (the tabs, new tab,
// close tab, paste, copy, select all, cancel, settings) into assets/images:
// thin white line drawings on a clear background, 81 by 81 pixels, the size
// BlackBerry 10 uses for action icons, after the look of the icons in its
// browser. They are drawn four times as large and scaled down, which smooths
// the slanted and round edges.
// Run from the repository's root: go run ./tools/icons
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

const (
	size  = 81 // the icon, in pixels
	scale = 4  // drawn this many times larger
	line  = 4  // the lines' thickness, in the icon's pixels
)

// icon is the large drawing: one byte of white per point.
type icon [size * scale][size * scale]bool

// box fills the rectangle x0, y0 to x1, y1 (in the icon's pixels, the end not
// included) with white, or clears it.
func (i *icon) box(x0, y0, x1, y1 float64, white bool) {
	for y := int(y0 * scale); y < int(y1*scale); y++ {
		for x := int(x0 * scale); x < int(x1*scale); x++ {
			if x >= 0 && y >= 0 && x < size*scale && y < size*scale {
				i[y][x] = white
			}
		}
	}
}

// frame draws the outline of a rectangle, the line inside it.
func (i *icon) frame(x0, y0, x1, y1 float64) {
	i.box(x0, y0, x1, y0+line, true)
	i.box(x0, y1-line, x1, y1, true)
	i.box(x0, y0, x0+line, y1, true)
	i.box(x1-line, y0, x1, y1, true)
}

// where fills every point for which in reports true (x and y in the icon's
// pixels).
func (i *icon) where(in func(x, y float64) bool) {
	for y := range i {
		for x := range i[y] {
			if in((float64(x)+0.5)/scale, (float64(y)+0.5)/scale) {
				i[y][x] = true
			}
		}
	}
}

// sheets draws two sheets, one in front of the other: the browser's tabs
// icon.
func sheets(i *icon) {
	i.frame(24, 10, 71, 57)
	i.box(10, 24, 57, 71, false)
	i.frame(10, 24, 57, 71)
}

func save(name string, img image.Image) {
	file, err := os.Create("assets/images/" + name + ".png")
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		log.Fatal(err)
	}
	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	draw := map[string]func(*icon){
		"tabs": sheets,
		"copy": sheets,
		// A sheet with a plus at its lower right corner: the browser's
		// new tab icon.
		"newtab": func(i *icon) {
			i.frame(10, 10, 57, 57)
			i.box(44, 44, 81, 81, false)
			i.box(58, 46, 58+line, 74, true)
			i.box(46, 58, 74, 58+line, true)
		},
		// A thin cross.
		"closetab": func(i *icon) {
			i.where(func(x, y float64) bool {
				if x < 24 || x > 57 || y < 24 || y > 57 {
					return false
				}
				d, e := x-y, x+y-81
				return (d > -2.2 && d < 2.2) || (e > -2.2 && e < 2.2)
			})
		},
		// A clipboard: the board and its clip.
		"paste": func(i *icon) {
			i.frame(16, 18, 65, 71)
			i.box(29, 10, 52, 24, true)
		},
		// A dashed frame around a filled square.
		"selectall": func(i *icon) {
			i.frame(10, 10, 71, 71)
			for _, at := range []float64{24, 38.5, 53} {
				i.box(at, 10, at+line, 71, false)
				i.box(10, at, 71, at+line, false)
			}
			i.box(29, 29, 52, 52, true)
		},
		// A ring with a stroke through it.
		"cancel": func(i *icon) {
			i.where(func(x, y float64) bool {
				d := (x-40.5)*(x-40.5) + (y-40.5)*(y-40.5)
				ring := d <= 30*30 && d >= (30-line)*(30-line)
				e := x + y - 81
				return ring || (d <= 30*30 && e > -2.8 && e < 2.8)
			})
		},
		// Three sliders: lines with a knob each.
		"settings": func(i *icon) {
			for n, knob := range []float64{24, 48, 32} {
				y := 20 + 20*float64(n)
				i.box(10, y, 71, y+line, true)
				i.box(knob, y-7, knob+10, y+line+7, true)
			}
		},
	}
	for name, f := range draw {
		var big icon
		f(&big)
		// Each pixel is as opaque as the share of its points that are
		// white.
		out := image.NewNRGBA(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				n := 0
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						if big[y*scale+dy][x*scale+dx] {
							n++
						}
					}
				}
				out.SetNRGBA(x, y, color.NRGBA{255, 255, 255, uint8(n * 255 / (scale * scale))})
			}
		}
		save(name, out)
	}
}
