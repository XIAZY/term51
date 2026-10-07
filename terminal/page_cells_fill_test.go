package terminal

import "testing"

// Go-only: ruling (pageCellsFill, the doubling copy that stands for @memset over cells, sets every cell and nothing else)
func TestPage_go_only_pageCellsFill_equals_a_loop(t *testing.T) {
	value := PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 'E'},
		style_id:    7,
		wide:        PageCellWideWide,
		protected:   true,
	}
	other := PageCell{content: PageCellContent{codepoint: 'x'}, hyperlink: true}
	for n := 0; n <= 300; n++ {
		// One guard cell on each side of the filled range.
		buf := make([]PageCell, n+2)
		for i := range buf {
			buf[i] = other
		}
		pageCellsFill(buf[1:1+n], value)
		if buf[0] != other || buf[n+1] != other {
			t.Fatalf("n=%d: a cell outside the range was written", n)
		}
		for i, c := range buf[1 : 1+n] {
			if c != value {
				t.Fatalf("n=%d: cell %d is %+v", n, i, c)
			}
		}
	}
}
