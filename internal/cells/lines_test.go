package cells

import (
	"image"
	"testing"
)

// edge is the coverage along one edge of a drawn cell: 'u', 'd', 'l' or 'r'.
func edge(img *image.Alpha, side byte) []uint8 {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	var out []uint8
	switch side {
	case 'u', 'd':
		y := 0
		if side == 'd' {
			y = h - 1
		}
		for x := 0; x < w; x++ {
			out = append(out, img.Pix[y*img.Stride+x])
		}
	default:
		x := 0
		if side == 'r' {
			x = w - 1
		}
		for y := 0; y < h; y++ {
			out = append(out, img.Pix[y*img.Stride+x])
		}
	}
	return out
}

// TestLinesMeetTheirNeighbours: at each edge of its cell a line character
// shows exactly the line it has in that direction (none, light, heavy or
// double), at the same place for every character, so that any two that
// belong together join without a gap or a step.
func TestLinesMeetTheirNeighbours(t *testing.T) {
	for _, size := range []image.Point{{18, 36}, {27, 53}, {9, 17}, {24, 48}} {
		w, h := size.X, size.Y
		th := max(w/8, 1)
		want := func(n int, weight byte) []uint8 {
			light, heavy, first, second := lineSpans(n, th)
			out := make([]uint8, n)
			set := func(s span) {
				for i := s.lo; i < s.hi; i++ {
					out[i] = 255
				}
			}
			switch weight {
			case '1':
				set(light)
			case '2':
				set(heavy)
			case '3':
				set(first)
				set(second)
			}
			return out
		}
		for r := rune(0x2500); r <= 0x257F; r++ {
			if !drawn(r) {
				continue
			}
			img := image.NewAlpha(image.Rect(0, 0, w, h))
			drawLines(img, r)
			arms := lineArms[(r-0x2500)*4:][:4]
			for i, side := range []byte("udlr") {
				n := w
				if side == 'l' || side == 'r' {
					n = h
				}
				if got, exp := edge(img, side), want(n, arms[i]); string(got) != string(exp) {
					t.Errorf("%c (U+%04X) at %dx%d, edge %c: got %v, want %v", r, r, w, h, side, got, exp)
				}
			}
		}
	}
}

func TestLineShapes(t *testing.T) {
	const w, h = 18, 36
	at := func(r rune) func(x, y int) bool {
		img := image.NewAlpha(image.Rect(0, 0, w, h))
		drawLines(img, r)
		return func(x, y int) bool { return img.Pix[y*img.Stride+x] != 0 }
	}
	th := max(w/8, 1)
	xl, _, xa, xb := lineSpans(w, th)
	yl, _, ya, yb := lineSpans(h, th)

	// ┌: nothing above or left of the corner, the corner itself filled.
	p := at('┌')
	if !p(xl.lo, yl.lo) || p(xl.lo, yl.lo-1) || p(xl.lo-1, yl.lo) || p(0, 0) || !p(w-1, yl.lo) || !p(xl.lo, h-1) {
		t.Error("┌ is not a corner")
	}
	// ╔: an outer and an inner corner with an empty channel between them.
	p = at('╔')
	if !p(xa.lo, ya.lo) || !p(xb.lo, yb.lo) || p(xl.lo, yl.lo) || p(xl.lo, h-1) || p(w-1, yl.lo) || p(xa.lo-1, ya.lo) || p(xb.lo, ya.hi) && p(xb.lo, yl.lo) {
		t.Error("╔ is not a double corner")
	}
	if p(xa.lo, ya.lo-1) || !p(xa.lo, h-1) || !p(xb.lo, h-1) || !p(w-1, ya.lo) || !p(w-1, yb.lo) {
		t.Error("╔: the lines do not run from the corners to the edges")
	}
	// ╠: the left line unbroken, the right one open towards the arm.
	p = at('╠')
	for y := 0; y < h; y++ {
		if !p(xa.lo, y) {
			t.Fatalf("╠: the left line is broken at row %d", y)
		}
	}
	if p(xb.lo, yl.lo) || !p(xb.lo, ya.lo) || !p(xb.lo, 0) {
		t.Error("╠: the right line is not open towards the arm")
	}
	// ╟: the light arm ends at the nearer line and leaves the channel empty.
	p = at('╟')
	if p(xl.lo, yl.lo) || !p(xb.lo, yl.lo) || !p(w-1, yl.lo) {
		t.Error("╟: the arm crosses into the double line")
	}
	// ╒: the light line closes the double line's end.
	p = at('╒')
	if !p(xl.lo, ya.lo) || !p(xl.lo, yb.lo) || !p(xl.lo, yl.lo) || p(xl.lo, ya.lo-1) || p(xl.lo-1, ya.lo) || p(xl.hi, yl.lo) {
		t.Error("╒: the light line does not close the double line")
	}
	// ╪: the light line crosses the double one.
	p = at('╪')
	for y := 0; y < h; y++ {
		if !p(xl.lo, y) {
			t.Fatalf("╪: the light line is broken at row %d", y)
		}
	}

	// Blocks.
	p = at('█')
	if !p(0, 0) || !p(w-1, h-1) {
		t.Error("█ is not full")
	}
	p = at('▄')
	if p(0, h/2-1) || !p(0, h/2) || !p(w-1, h-1) {
		t.Error("▄ is not the lower half")
	}
	p = at('▌')
	if !p(0, 0) || !p(w/2-1, h-1) || p(w/2, 0) {
		t.Error("▌ is not the left half")
	}
	p = at('▚')
	if !p(0, 0) || p(w-1, 0) || p(0, h-1) || !p(w-1, h-1) {
		t.Error("▚ is not upper left and lower right")
	}
	img := image.NewAlpha(image.Rect(0, 0, w, h))
	drawLines(img, '▒')
	if img.Pix[0] != 128 {
		t.Errorf("▒ has coverage %d, want 128", img.Pix[0])
	}
}

func TestSheetDrawsLinesItself(t *testing.T) {
	s := newSheet(t, 28)
	w, h := s.CellSize()
	n := s.slot([]rune{'│'}, 0, false)
	r := s.rect(int(n))
	for y := 0; y < h; y++ {
		if s.Pix[(r.Min.Y+y)*SheetSize+r.Min.X+w/2] != 255 {
			t.Fatalf("│ does not reach row %d of its cell", y)
		}
	}
	// A diagonal stays the font's.
	if d := s.slot([]rune{'╱'}, 0, false); s.ink(d) == 0 {
		t.Error("╱ is empty")
	}
}
