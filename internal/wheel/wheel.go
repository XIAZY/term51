// Package wheel encodes a turn of the mouse wheel the way a terminal reports
// it to a program that asked for mouse events, in each of the five formats
// a program can choose. The rules are upstream Ghostty's
// (src/input/mouse_encode.zig): a wheel turn is a press of button 4 (up) or
// 5 (down), button codes 64 and 65, with no modifiers.
package wheel

import (
	"fmt"
	"unicode/utf8"
)

// Format is how a mouse event is written.
type Format int

const (
	X10       Format = iota // CSI M Cb Cx Cy, each byte 32 more than its value
	UTF8                    // as X10, with the coordinates as UTF-8 characters
	SGR                     // CSI < Cb ; x ; y M
	URXVT                   // CSI Cb+32 ; x ; y M
	SGRPixels               // as SGR, with the position in pixels
)

// Button is the wheel's direction.
type Button byte

const (
	Up   Button = 64 // button 4: towards older content
	Down Button = 65 // button 5
)

// Encode gives the report of one wheel turn b at the cell col, row (0-based)
// and the pixel px, py inside the terminal. It is nil where the format
// cannot say the position: X10 stops at the 223rd column and row.
func Encode(f Format, b Button, col, row, px, py int) []byte {
	switch f {
	case X10:
		if col > 222 || row > 222 {
			return nil
		}
		return []byte{0x1b, '[', 'M', 32 + byte(b), 32 + byte(col) + 1, 32 + byte(row) + 1}
	case UTF8:
		out := []byte{0x1b, '[', 'M', 32 + byte(b)}
		out = utf8.AppendRune(out, rune(col+33))
		return utf8.AppendRune(out, rune(row+33))
	case SGR:
		return fmt.Appendf(nil, "\x1b[<%d;%d;%dM", b, col+1, row+1)
	case URXVT:
		return fmt.Appendf(nil, "\x1b[%d;%d;%dM", 32+int(b), col+1, row+1)
	case SGRPixels:
		return fmt.Appendf(nil, "\x1b[<%d;%d;%dM", b, px, py)
	}
	return nil
}
