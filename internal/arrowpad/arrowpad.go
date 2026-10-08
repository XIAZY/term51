// Package arrowpad turns a finger's swipe on the arrow pad into arrow keys.
//
// The pad appears when the terminal is pressed and held. From then on the
// finger works like a joystick whose middle is where it went down: moved
// past a threshold in one of the four directions it sends that arrow key
// once, and held out there it repeats the key, faster the further out it
// is. Back inside the threshold it stops.
//
// This package is the rule only: it is told the finger's offset and the
// time, and says which key to send and when to ask again. The page draws
// the pad and reports the finger; the app sends the keys.
package arrowpad

import "time"

const (
	// Threshold is how far, in pixels, the finger must be from where it
	// went down for an arrow key to be sent (about 3.5 mm on the Passport).
	Threshold = 64.0
	// Axis is how many times stronger the other axis must be before the
	// direction changes from one axis to the other: a swipe that is not
	// quite straight keeps its direction.
	Axis = 1.5
	// repeatDelay is how long the finger stays past the threshold before
	// the key repeats.
	repeatDelay = 350 * time.Millisecond
	// The key repeats slowRate times a second at the threshold and fastRate
	// times a second at fastAt thresholds and beyond, and in proportion in
	// between.
	slowRate = 8.0
	fastRate = 25.0
	fastAt   = 3.0
)

// Pad is the state of one press: zero when the finger has just gone down.
type Pad struct {
	dir   string    // the direction being sent, "" inside the threshold
	since time.Time // when its first key was sent
	last  time.Time // when its last key was sent
}

// direction is the arrow the offset dx, dy (x to the right, y down) points
// to and how far along it the finger is. An offset between two arrows goes
// to the stronger axis; while a direction is being sent, the other axis
// takes over only when it is Axis times stronger.
func (p *Pad) direction(dx, dy float64) (string, float64) {
	ax, ay := abs(dx), abs(dy)
	horizontal := ax >= ay
	switch p.dir {
	case "left", "right":
		horizontal = ay < Axis*ax
	case "up", "down":
		horizontal = ax >= Axis*ay
	}
	switch {
	case horizontal && dx < 0:
		return "left", ax
	case horizontal:
		return "right", ax
	case dy < 0:
		return "up", ay
	}
	return "down", ay
}

// Direction is the arrow the offset points to and how full it is: 0 at the
// middle, 1 at the threshold and beyond. It is for drawing the pad.
func (p *Pad) Direction(dx, dy float64) (dir string, fill float64) {
	dir, dist := p.direction(dx, dy)
	return dir, min(dist/Threshold, 1)
}

// Update is called with the finger's offset from where it went down, when
// it moves and when the time Update asked for has passed. It returns the
// arrow key to send now ("up", "down", "left", "right", or "" for none) and
// how long from now to call it again if the finger has not moved by then (0:
// no need).
func (p *Pad) Update(dx, dy float64, now time.Time) (key string, again time.Duration) {
	dir, dist := p.direction(dx, dy)
	if dist < Threshold {
		p.dir = ""
		return "", 0
	}
	if dir != p.dir {
		// Past the threshold in a new direction: its key once, at once.
		p.dir, p.since, p.last = dir, now, now
		return dir, repeatDelay
	}
	if wait := p.since.Add(repeatDelay).Sub(now); wait > 0 {
		return "", wait
	}
	far := min((dist/Threshold-1)/(fastAt-1), 1)
	interval := time.Duration(float64(time.Second) / (slowRate + (fastRate-slowRate)*far))
	// The first repeat comes when the delay is over; the following ones an
	// interval after the last.
	if p.last != p.since {
		if wait := p.last.Add(interval).Sub(now); wait > 0 {
			return "", wait
		}
	}
	p.last = now
	return dir, interval
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
