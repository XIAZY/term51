// Command icons draws the pictures the system has none for into
// assets/images: the menu icons (close tab, paste, copy, select all, cancel,
// settings), the signs of the three keys on the system's bar (Esc, Ctrl,
// Tab) and the arrow pad's four arrows, as thin white line drawings on
// a clear background, 81 by 81 pixels, the size BlackBerry 10 uses for
// action icons and after the look of the icons in its browser; and the arrow
// pad's ground. They are drawn four times as large and scaled down, which
// smooths the slanted and round edges.
// Run from the repository's root: go run ./tools/icons
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
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

// arrow draws an arrow from the icon's middle outwards in the direction dx,
// dy (one of them 0, the other 1 or -1): a shaft and a head. The arrow pad
// shows the four of them, and colours them itself.
func arrow(dx, dy float64) func(*icon) {
	return func(i *icon) {
		i.where(func(x, y float64) bool {
			// Along the arrow (a) and across it (b), from the middle.
			x, y = x-40.5, y-40.5
			a, b := x*dx+y*dy, math.Abs(x*dy-y*dx)
			shaft := a >= -26 && a <= 24 && b <= line/2.0+0.5
			// The head: two strokes meeting at the tip.
			d := math.Abs((26-a)-b) / math.Sqrt2
			head := a >= 4 && a <= 26.5 && b <= 22 && d <= line/2.0+0.5
			return shaft || head
		})
	}
}

// stroke reports whether the point x, y is on the line from x0, y0 to x1, y1,
// drawn as thick as the icons' lines, with round ends.
func stroke(x, y, x0, y0, x1, y1 float64) bool {
	dx, dy := x1-x0, y1-y0
	t := math.Max(0, math.Min(1, ((x-x0)*dx+(y-y0)*dy)/(dx*dx+dy*dy)))
	px, py := x0+t*dx-x, y0+t*dy-y
	return px*px+py*py <= (line/2.0+0.4)*(line/2.0+0.4)
}

// The three keys on the system's bar. Each is a sign, not its name: the
// name is the button's title.

// escape draws the escape sign: a ring, and an arrow leaving it up and to
// the left.
func escape(i *icon) {
	i.where(func(x, y float64) bool {
		d := (x-48)*(x-48) + (y-48)*(y-48)
		ring := d <= 21*21 && d >= (21-line)*(21-line)
		// The ring is open where the arrow leaves it.
		if ring && x < 42 && y < 42 {
			ring = false
		}
		return ring || stroke(x, y, 46, 46, 14, 14) || stroke(x, y, 14, 14, 14, 32) || stroke(x, y, 14, 14, 32, 14)
	})
}

// control draws the caret that stands for the Control key.
func control(i *icon) {
	i.where(func(x, y float64) bool {
		return stroke(x, y, 40.5, 24, 18, 50) || stroke(x, y, 40.5, 24, 63, 50)
	})
}

// tabKey draws an arrow running into a bar.
func tabKey(i *icon) {
	i.where(func(x, y float64) bool {
		return stroke(x, y, 12, 40.5, 56, 40.5) || stroke(x, y, 56, 40.5, 42, 26.5) || stroke(x, y, 56, 40.5, 42, 54.5) ||
			stroke(x, y, 67, 22, 67, 59)
	})
}

// padSize is the arrow pad's side in pixels.
const padSize = 300

// pad draws the arrow pad's ground: a dark square with round corners and a
// thin lighter rim, a little see-through, as large as the pad, with a grey
// dot in the middle where the finger's swipe starts from.
func pad() image.Image {
	const radius, rim = 36.0, 2.0
	// inside reports how much of the pixel x, y lies in the square with
	// round corners that is inset by d.
	inside := func(x, y int, d float64) float64 {
		n := 0
		for sy := 0; sy < scale; sy++ {
			for sx := 0; sx < scale; sx++ {
				px, py := float64(x)+(float64(sx)+0.5)/scale, float64(y)+(float64(sy)+0.5)/scale
				// The distance to the nearest corner's centre, where the
				// point is in a corner.
				cx := math.Max(math.Max(d+radius-px, px-(padSize-d-radius)), 0)
				cy := math.Max(math.Max(d+radius-py, py-(padSize-d-radius)), 0)
				if px >= d && py >= d && px <= padSize-d && py <= padSize-d && cx*cx+cy*cy <= radius*radius {
					n++
				}
			}
		}
		return float64(n) / (scale * scale)
	}
	out := image.NewNRGBA(image.Rect(0, 0, padSize, padSize))
	for y := 0; y < padSize; y++ {
		for x := 0; x < padSize; x++ {
			whole, core := inside(x, y, 0), inside(x, y, rim)
			// The rim is #5a5a5a, the ground #262626; 235 of 255 opaque.
			grey := 0x26 + (0x5a-0x26)*(whole-core)/math.Max(whole, 1.0/255)
			if dx, dy := float64(x)+0.5-padSize/2, float64(y)+0.5-padSize/2; dx*dx+dy*dy <= 6.5*6.5 {
				grey = 0x80
			}
			out.SetNRGBA(x, y, color.NRGBA{uint8(grey), uint8(grey), uint8(grey), uint8(235 * whole)})
		}
	}
	return out
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
	save("pad", pad())
	draw := map[string]func(*icon){
		"key-esc":     escape,
		"key-ctrl":    control,
		"key-tab":     tabKey,
		"arrow-left":  arrow(-1, 0),
		"arrow-right": arrow(1, 0),
		"arrow-up":    arrow(0, -1),
		"arrow-down":  arrow(0, 1),
		"tabs":        sheets,
		"copy":        sheets,
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
		save(name, render(f, color.NRGBA{255, 255, 255, 255}))
	}
	// Ctrl once more in BlackBerry blue, for while it is armed: the bar
	// shows an action's picture as it is and cannot colour it.
	save("key-ctrl-armed", render(control, color.NRGBA{0x00, 0xa8, 0xdf, 255}))
}

// render draws an icon in the colour c: each pixel is as opaque as the share
// of its points that the drawing covers.
func render(f func(*icon), c color.NRGBA) image.Image {
	var big icon
	f(&big)
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
			out.SetNRGBA(x, y, color.NRGBA{c.R, c.G, c.B, uint8(n * 255 / (scale * scale))})
		}
	}
	return out
}
