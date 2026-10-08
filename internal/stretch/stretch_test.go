package stretch

import (
	"testing"
	"time"
)

// hold keeps the finger at row y of a 10-row terminal for d, calling Update
// whenever it asked to be called, and returns the rows scrolled in all.
func hold(e *Edge, y float64, d time.Duration) (rows int) {
	start := time.Unix(1000, 0)
	for now := time.Duration(0); now <= d; {
		n, again := e.Update(y, 10, start.Add(now))
		rows += n
		if again == 0 {
			break
		}
		now += again
	}
	return rows
}

func TestEdge(t *testing.T) {
	for _, c := range []struct {
		y    float64
		rate float64 // rows a second; negative: back into the scrollback
	}{
		{5, 0}, {1.5, 0}, {8.5, 0}, // not in a zone
		{1.49, -4}, {8.51, 4}, // just inside: slow
		{0.75, -17}, {9.25, 17}, // half way in
		{0, -30}, {10, 30}, {-3, -30}, {40, 30}, // at the edge and beyond: fast
	} {
		var e Edge
		got := float64(hold(&e, c.y, 2*time.Second)) / 2
		if got < c.rate-1.1 || got > c.rate+1.1 {
			t.Errorf("finger at row %v: %.1f rows a second, want about %v", c.y, got, c.rate)
		}
	}
}

// The first row is scrolled at once when the finger enters a zone; out of
// the zone it stops, and nothing more is asked for.
func TestEdgeStartsAtOnceAndStops(t *testing.T) {
	var e Edge
	start := time.Unix(1000, 0)
	if n, again := e.Update(0.5, 10, start); n != -1 || again <= 0 {
		t.Errorf("entering the top zone: scroll %d, again %v", n, again)
	}
	if n, again := e.Update(0.5, 10, start.Add(time.Millisecond)); n != 0 || again <= 0 {
		t.Errorf("a move a moment later: scroll %d, again %v", n, again)
	}
	if n, again := e.Update(5, 10, start.Add(2*time.Millisecond)); n != 0 || again != 0 {
		t.Errorf("out of the zone: scroll %d, again %v", n, again)
	}
	if n, _ := e.Update(9.9, 10, start.Add(3*time.Millisecond)); n != 1 {
		t.Errorf("entering the bottom zone: scroll %d", n)
	}
}
