package selbar

import "testing"

func TestPlace(t *testing.T) {
	// The phone's view at the default size: 80 columns of 18 pixels, 30
	// rows of 40.
	const cols, rows, cw, ch = 80, 30, 18, 40
	for _, c := range []struct {
		name        string
		first, last Row
		seen        bool
		x, y        int
	}{
		{"above a word, centred over it", Row{10, 40, 49}, Row{10, 40, 49}, true, 45*18 - 224, 10*40 - 20 - 104},
		{"above the first of several rows", Row{10, 40, 79}, Row{12, 0, 5}, true, 60*18 - 224, 10*40 - 20 - 104},
		{"kept inside the left side", Row{10, 0, 3}, Row{10, 0, 3}, true, 8, 10*40 - 20 - 104},
		{"kept inside the right side", Row{10, 76, 79}, Row{10, 76, 79}, true, 80*18 - 448 - 8, 10*40 - 20 - 104},
		{"the lowest row that still has room above", Row{4, 40, 49}, Row{4, 40, 49}, true, 45*18 - 224, 36},
		{"no room above: below the last row, over its part", Row{2, 40, 79}, Row{4, 0, 9}, true, 8, 5*40 + 160},
		{"room for neither: at the top", Row{2, 40, 79}, Row{24, 0, 9}, true, 40*18 - 224, 20},
		{"the whole view selected: at the top", Row{0, 0, 79}, Row{29, 0, 79}, true, 40*18 - 224, 20},
		{"none of it in the view: at the top", Row{}, Row{}, false, 40*18 - 224, 20},
	} {
		if x, y := Place(c.first, c.last, c.seen, cols, rows, cw, ch); x != c.x || y != c.y {
			t.Errorf("%s: at %d, %d, want %d, %d", c.name, x, y, c.x, c.y)
		}
	}
}
