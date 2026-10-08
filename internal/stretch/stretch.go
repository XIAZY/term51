// Package stretch is the rule for scrolling while a selection is being
// stretched: a finger held near the top or the bottom edge of the terminal
// scrolls the screen by itself, faster the nearer it is to the edge, so that
// the selection can run into text that is not on screen.
//
// It is the rule only, as internal/arrowpad is for the arrow pad: it is told
// where the finger is and the time, and says how many rows to scroll now
// and when to ask again. The session moves the selection; the app's loop
// does the scrolling.
package stretch

import "time"

const (
	// zone is how near an edge, in rows, the finger must be for the screen
	// to scroll.
	zone = 1.5
	// The screen scrolls slowRate rows a second with the finger at the inner
	// side of the zone and fastRate rows a second with it at the edge
	// itself, and in proportion in between.
	slowRate = 4.0
	fastRate = 30.0
)

// Edge is the state of one stretch: zero when the finger has just come down.
type Edge struct {
	on   bool      // the finger is in a zone
	last time.Time // when a row was last scrolled
}

// Update is called with the finger's place, in rows from the top of a
// terminal of the given number of rows (2.5 is the middle of the third row),
// when the finger moves and when the time Update asked for has passed. It
// returns how many rows to scroll now (negative: back into the scrollback;
// positive: forward; 0: none) and how long from now to call it again if the
// finger has not moved by then (0: no need).
func (e *Edge) Update(y float64, rows int, now time.Time) (scroll int, again time.Duration) {
	// How deep in a zone the finger is: 0 at its inner side, 1 at the edge.
	depth, dir := 0.0, 0
	switch {
	case y < zone:
		depth, dir = (zone-y)/zone, -1
	case y > float64(rows)-zone:
		depth, dir = (y-(float64(rows)-zone))/zone, 1
	}
	if dir == 0 {
		e.on = false
		return 0, 0
	}
	interval := time.Duration(float64(time.Second) / (slowRate + (fastRate-slowRate)*min(depth, 1)))
	if e.on {
		if wait := e.last.Add(interval).Sub(now); wait > 0 {
			return 0, wait
		}
	}
	e.on, e.last = true, now
	return dir, interval
}
