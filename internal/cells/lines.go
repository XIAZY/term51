package cells

import "image"

// The line-drawing and block characters (U+2500 to U+259F) are drawn here
// and not taken from the font: a font's own do not reach the edges of our
// cell, so the frames of full-screen programs would show gaps.

// lineArms says, for U+2500 to U+257F, what each character has towards the
// top, bottom, left and right: nothing, a light line (1), a heavy line (2)
// or a double line (3). Dashed lines are drawn solid, rounded corners
// square. The three diagonals have "xxxx" and stay the font's.
const lineArms = "" +
	"0011" + "0022" + "1100" + "2200" + "0011" + "0022" + "1100" + "2200" + // 2500 ─ ━ │ ┃ ┄ ┅ ┆ ┇
	"0011" + "0022" + "1100" + "2200" + "0101" + "0102" + "0201" + "0202" + // 2508 ┈ ┉ ┊ ┋ ┌ ┍ ┎ ┏
	"0110" + "0120" + "0210" + "0220" + "1001" + "1002" + "2001" + "2002" + // 2510 ┐ ┑ ┒ ┓ └ ┕ ┖ ┗
	"1010" + "1020" + "2010" + "2020" + "1101" + "1102" + "2101" + "1201" + // 2518 ┘ ┙ ┚ ┛ ├ ┝ ┞ ┟
	"2201" + "2102" + "1202" + "2202" + "1110" + "1120" + "2110" + "1210" + // 2520 ┠ ┡ ┢ ┣ ┤ ┥ ┦ ┧
	"2210" + "2120" + "1220" + "2220" + "0111" + "0121" + "0112" + "0122" + // 2528 ┨ ┩ ┪ ┫ ┬ ┭ ┮ ┯
	"0211" + "0221" + "0212" + "0222" + "1011" + "1021" + "1012" + "1022" + // 2530 ┰ ┱ ┲ ┳ ┴ ┵ ┶ ┷
	"2011" + "2021" + "2012" + "2022" + "1111" + "1121" + "1112" + "1122" + // 2538 ┸ ┹ ┺ ┻ ┼ ┽ ┾ ┿
	"2111" + "1211" + "2211" + "2121" + "2112" + "1221" + "1212" + "2122" + // 2540 ╀ ╁ ╂ ╃ ╄ ╅ ╆ ╇
	"1222" + "2221" + "2212" + "2222" + "0011" + "0022" + "1100" + "2200" + // 2548 ╈ ╉ ╊ ╋ ╌ ╍ ╎ ╏
	"0033" + "3300" + "0103" + "0301" + "0303" + "0130" + "0310" + "0330" + // 2550 ═ ║ ╒ ╓ ╔ ╕ ╖ ╗
	"1003" + "3001" + "3003" + "1030" + "3010" + "3030" + "1103" + "3301" + // 2558 ╘ ╙ ╚ ╛ ╜ ╝ ╞ ╟
	"3303" + "1130" + "3310" + "3330" + "0133" + "0311" + "0333" + "1033" + // 2560 ╠ ╡ ╢ ╣ ╤ ╥ ╦ ╧
	"3011" + "3033" + "1133" + "3311" + "3333" + "0101" + "0110" + "1010" + // 2568 ╨ ╩ ╪ ╫ ╬ ╭ ╮ ╯
	"1001" + "xxxx" + "xxxx" + "xxxx" + "0010" + "1000" + "0001" + "0100" + // 2570 ╰ ╱ ╲ ╳ ╴ ╵ ╶ ╷
	"0020" + "2000" + "0002" + "0200" + "0012" + "1200" + "0021" + "2100" //   2578 ╸ ╹ ╺ ╻ ╼ ╽ ╾ ╿

// drawn reports whether the character is one this file draws.
func drawn(r rune) bool {
	if r >= 0x2500 && r <= 0x257F {
		return lineArms[(r-0x2500)*4] != 'x'
	}
	return r >= 0x2580 && r <= 0x259F
}

// span is a range of pixels along one axis.
type span struct{ lo, hi int }

// lineSpans are, across a cell of the given size, the places of a light
// line, a heavy line, and the two lines of a double line.
func lineSpans(size, t int) (light, heavy, first, second span) {
	light = span{size/2 - t/2, size/2 - t/2 + t}
	heavy = span{light.lo - (t+1)/2, light.lo - (t+1)/2 + 2*t}
	first = span{light.lo - t, light.lo}
	second = span{light.hi, light.hi + t}
	return
}

// drawLines draws a character of this file into img, a cell with its top left
// corner at the origin.
func drawLines(img *image.Alpha, r rune) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	fill := func(x0, y0, x1, y1 int, v uint8) {
		for y := max(y0, 0); y < min(y1, h); y++ {
			for x := max(x0, 0); x < min(x1, w); x++ {
				img.Pix[y*img.Stride+x] = v
			}
		}
	}
	if r >= 0x2580 {
		block(r, w, h, fill)
		return
	}

	arms := lineArms[(r-0x2500)*4:][:4]
	up, down, left, right := int(arms[0]-'0'), int(arms[1]-'0'), int(arms[2]-'0'), int(arms[3]-'0')
	t := max(w/8, 1)
	xl, xh, xa, xb := lineSpans(w, t)
	yl, yh, ya, yb := lineSpans(h, t)

	// A double line is drawn as a band three lines thick with its middle
	// taken out again. Where another double line meets it, the band goes on
	// to that one's far line, which makes the outer corner; the middles
	// taken out of both make the inner one.
	vDouble, hDouble := up == 3 || down == 3, left == 3 || right == 3
	bandX0, bandX1 := xl.lo, xl.hi // how far a horizontal band goes into the middle
	if vDouble {
		bandX0, bandX1 = xa.lo, xb.hi
	}
	bandY0, bandY1 := yl.lo, yl.hi
	if hDouble {
		bandY0, bandY1 = ya.lo, yb.hi
	}
	if left == 3 {
		fill(0, ya.lo, bandX1, yb.hi, 255)
	}
	if right == 3 {
		fill(bandX0, ya.lo, w, yb.hi, 255)
	}
	if up == 3 {
		fill(xa.lo, 0, xb.hi, bandY1, 255)
	}
	if down == 3 {
		fill(xa.lo, bandY0, xb.hi, h, 255)
	}
	if left == 3 {
		fill(0, yl.lo, xl.hi, yl.hi, 0)
	}
	if right == 3 {
		fill(xl.lo, yl.lo, w, yl.hi, 0)
	}
	if up == 3 {
		fill(xl.lo, 0, xl.hi, yl.hi, 0)
	}
	if down == 3 {
		fill(xl.lo, yl.lo, xl.hi, h, 0)
	}

	// A light or heavy line runs from the edge to the middle. At a double
	// line it does one of three things: it crosses it when it goes on on the
	// other side; it stops at the nearer of the two lines when the double
	// line passes by; and it closes the double line's end when that ends
	// here.
	arm := func(weight, opposite, sideA, sideB int, horizontal, towardsStart bool) {
		if weight != 1 && weight != 2 {
			return
		}
		// across is the line's place across its direction, mid, first and
		// second what it meets in the middle of the cell along it.
		across, mid, first, second, size := yl, xl, xa, xb, w
		if weight == 2 {
			across, mid = yh, xh
		}
		if !horizontal {
			across, mid, first, second, size = xl, yl, ya, yb, h
			if weight == 2 {
				across, mid = xh, yh
			}
		}
		lo, hi := mid.lo, size
		if towardsStart {
			lo, hi = 0, mid.hi
		}
		if (sideA == 3 || sideB == 3) && opposite != 1 && opposite != 2 {
			passes := sideA == 3 && sideB == 3
			switch {
			case towardsStart && passes:
				hi = first.hi
			case towardsStart:
				hi = second.hi
			case passes:
				lo = second.lo
			default:
				lo = first.lo
			}
		}
		if horizontal {
			fill(lo, across.lo, hi, across.hi, 255)
		} else {
			fill(across.lo, lo, across.hi, hi, 255)
		}
	}
	arm(left, right, up, down, true, true)
	arm(right, left, up, down, true, false)
	arm(up, down, left, right, false, true)
	arm(down, up, left, right, false, false)
}

// block draws a block element, U+2580 to U+259F.
func block(r rune, w, h int, fill func(x0, y0, x1, y1 int, v uint8)) {
	switch {
	case r == 0x2580: // upper half
		fill(0, 0, w, h/2, 255)
	case r <= 0x2588: // lower one eighth to full
		fill(0, h-h*int(r-0x2580)/8, w, h, 255)
	case r <= 0x258F: // left seven eighths to one eighth
		fill(0, 0, w*int(0x2590-r)/8, h, 255)
	case r == 0x2590: // right half
		fill(w/2, 0, w, h, 255)
	case r <= 0x2593: // light, medium and dark shade
		fill(0, 0, w, h, uint8(64*int(r-0x2590)))
	case r == 0x2594: // upper one eighth
		fill(0, 0, w, h/8, 255)
	case r == 0x2595: // right one eighth
		fill(w-w/8, 0, w, h, 255)
	default:
		// The quadrants, U+2596 to U+259F: which of upper left, upper
		// right, lower left and lower right are filled.
		const ul, ur, ll, lr = 1, 2, 4, 8
		q := [...]int{ll, lr, ul, ul | ll | lr, ul | lr, ul | ur | ll, ul | ur | lr, ur, ur | ll, ur | ll | lr}[r-0x2596]
		if q&ul != 0 {
			fill(0, 0, w/2, h/2, 255)
		}
		if q&ur != 0 {
			fill(w/2, 0, w, h/2, 255)
		}
		if q&ll != 0 {
			fill(0, h/2, w/2, h, 255)
		}
		if q&lr != 0 {
			fill(w/2, h/2, w, h, 255)
		}
	}
}
