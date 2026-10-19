package wheel

import "testing"

func TestEncode(t *testing.T) {
	cases := []struct {
		name     string
		f        Format
		b        Button
		col, row int
		px, py   int
		want     string
	}{
		{"x10 up", X10, Up, 4, 9, 0, 0, "\x1b[M\x60\x25\x2a"},
		{"x10 down", X10, Down, 0, 0, 0, 0, "\x1b[M\x61\x21\x21"},
		{"x10 last column and row", X10, Up, 222, 222, 0, 0, "\x1b[M\x60\xff\xff"},
		{"x10 beyond", X10, Up, 223, 0, 0, 0, ""},
		{"x10 beyond in rows", X10, Down, 0, 223, 0, 0, ""},
		{"utf8 small", UTF8, Up, 4, 9, 0, 0, "\x1b[M\x60\x25\x2a"},
		{"utf8 above 127", UTF8, Down, 200, 1, 0, 0, "\x1b[M\x61" + string(rune(233)) + "\x22"},
		{"utf8 large", UTF8, Up, 2014, 300, 0, 0, "\x1b[M\x60" + string(rune(2047)) + string(rune(333))},
		{"sgr up", SGR, Up, 4, 9, 0, 0, "\x1b[<64;5;10M"},
		{"sgr down", SGR, Down, 79, 23, 0, 0, "\x1b[<65;80;24M"},
		{"urxvt up", URXVT, Up, 4, 9, 0, 0, "\x1b[96;5;10M"},
		{"urxvt down", URXVT, Down, 0, 0, 0, 0, "\x1b[97;1;1M"},
		{"sgr pixels up", SGRPixels, Up, 4, 9, 37, 95, "\x1b[<64;37;95M"},
		{"sgr pixels down", SGRPixels, Down, 0, 0, 0, 0, "\x1b[<65;0;0M"},
	}
	for _, c := range cases {
		if got := string(Encode(c.f, c.b, c.col, c.row, c.px, c.py)); got != c.want {
			t.Errorf("%s: Encode = %q, want %q", c.name, got, c.want)
		}
	}
}
