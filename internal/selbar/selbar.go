// Package selbar decides where the selection's bar goes: the small strip of
// buttons (Copy, Paste, Select Line, Select All) that floats by the selected
// text. The page draws it; the place is worked out here, where it can be
// tested without the phone.
package selbar

const (
	Width  = 448 // the bar, in pixels: four cells of 112
	Height = 104
	Gap    = 20 // between the bar and the selected text
	Margin = 8  // between the bar and the terminal's sides
)

// Row is the selected part of one row of the view: the row, and its first
// and last selected columns.
type Row struct{ Y, Left, Right int }

// Place gives the bar's top left corner in the terminal, in pixels. first
// and last are the first and the last row of the view with selected text in
// them. The bar goes above the first, centred over its selected part; below
// the last where there is no room above; and at the top of the terminal
// where there is room for neither (the selection fills the view) or where
// none of the selection is in the view (seen is false). It never leaves the
// terminal's sides. cols and rows are the view in cells, cw and ch a cell in
// pixels.
func Place(first, last Row, seen bool, cols, rows, cw, ch int) (x, y int) {
	centre := func(r Row) int {
		x := (r.Left+r.Right+1)*cw/2 - Width/2
		return max(Margin, min(x, cols*cw-Width-Margin))
	}
	if !seen {
		return centre(Row{0, 0, cols - 1}), Gap
	}
	if y := first.Y*ch - Gap - Height; y >= 0 {
		return centre(first), y
	}
	if y := (last.Y+1)*ch + Gap; y+Height <= rows*ch {
		return centre(last), y
	}
	return centre(Row{0, 0, cols - 1}), Gap
}
