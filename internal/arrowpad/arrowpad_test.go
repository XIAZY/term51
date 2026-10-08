package arrowpad

import (
	"testing"
	"time"
)

// run moves the finger to dx, dy at time zero and keeps it there for d,
// calling Update whenever it asked to be called, as the app does. It
// returns the keys sent and when.
func run(p *Pad, dx, dy float64, d time.Duration) (keys []string, at []time.Duration) {
	start := time.Unix(1000, 0)
	for now := time.Duration(0); now <= d; {
		key, again := p.Update(dx, dy, start.Add(now))
		if key != "" {
			keys = append(keys, key)
			at = append(at, now)
		}
		if again == 0 {
			break
		}
		now += again
	}
	return keys, at
}

func TestThreshold(t *testing.T) {
	for _, c := range []struct {
		dx, dy float64
		want   string
	}{
		{0, 0, ""}, {63, 0, ""}, {0, -63.9, ""}, {40, 40, ""},
		{64, 0, "right"}, {-64, 0, "left"}, {0, 64, "down"}, {0, -64, "up"},
		{100, 30, "right"}, {-30, -100, "up"},
	} {
		var p Pad
		keys, _ := run(&p, c.dx, c.dy, 0)
		got := ""
		if len(keys) > 0 {
			got = keys[0]
		}
		if got != c.want || len(keys) > 1 {
			t.Errorf("finger at %v, %v: keys %v, want %q once", c.dx, c.dy, keys, c.want)
		}
	}
}

// Held past the threshold, the key is sent once, then after the delay it
// repeats: 8 times a second at the threshold, 25 at three thresholds and
// beyond.
func TestRepeat(t *testing.T) {
	for _, c := range []struct {
		dist float64
		rate float64 // repeats a second once repeating
	}{{64, 8}, {128, 16.5}, {192, 25}, {500, 25}} {
		var p Pad
		keys, at := run(&p, c.dist, 0, 2350*time.Millisecond)
		if len(keys) < 3 || at[0] != 0 || at[1] != 350*time.Millisecond {
			t.Fatalf("at %v pixels: keys at %v; want the first at once and the second after 350 ms", c.dist, at)
		}
		// Two seconds of repeating after the delay.
		if got, want := float64(len(keys)-2)/2, c.rate; got < want-0.6 || got > want+0.6 {
			t.Errorf("at %v pixels: %.1f repeats a second, want about %.1f", c.dist, got, want)
		}
		for _, k := range keys {
			if k != "right" {
				t.Fatalf("at %v pixels: keys %v", c.dist, keys)
			}
		}
	}
}

// Back inside the threshold it stops, and out again it starts afresh: one
// key at once, and the delay before repeating.
func TestBackInsideStops(t *testing.T) {
	var p Pad
	start := time.Unix(1000, 0)
	if key, _ := p.Update(0, -80, start); key != "up" {
		t.Fatalf("out: %q", key)
	}
	if key, again := p.Update(0, -30, start.Add(100*time.Millisecond)); key != "" || again != 0 {
		t.Errorf("back inside: key %q, again %v", key, again)
	}
	if key, again := p.Update(0, -80, start.Add(200*time.Millisecond)); key != "up" || again != 350*time.Millisecond {
		t.Errorf("out again: key %q, again %v", key, again)
	}
	if key, _ := p.Update(0, -80, start.Add(300*time.Millisecond)); key != "" {
		t.Errorf("a move during the delay sent %q", key)
	}
}

// The direction changes to the other axis only when that axis is clearly
// stronger, one and a half times; then the new key is sent at once.
func TestTheOtherAxisMustClearlyWin(t *testing.T) {
	var p Pad
	start := time.Unix(1000, 0)
	p.Update(100, 0, start)
	if key, _ := p.Update(100, 120, start.Add(10*time.Millisecond)); key != "" {
		t.Errorf("down is stronger but not one and a half times: sent %q", key)
	}
	if dir, _ := p.Direction(100, 120); dir != "right" {
		t.Errorf("the pad shows %q, want right", dir)
	}
	if key, _ := p.Update(100, 151, start.Add(20*time.Millisecond)); key != "down" {
		t.Errorf("down is one and a half times stronger: sent %q", key)
	}
	// And the same rule back.
	if key, _ := p.Update(200, 151, start.Add(30*time.Millisecond)); key != "" {
		t.Errorf("right is stronger but not one and a half times: sent %q", key)
	}
}

func TestDirectionForDrawing(t *testing.T) {
	var p Pad
	for _, c := range []struct {
		dx, dy float64
		dir    string
		fill   float64
	}{{32, 0, "right", 0.5}, {0, -16, "up", 0.25}, {-640, 3, "left", 1}, {0, 64, "down", 1}} {
		if dir, fill := p.Direction(c.dx, c.dy); dir != c.dir || fill != c.fill {
			t.Errorf("Direction(%v, %v) = %q, %v; want %q, %v", c.dx, c.dy, dir, fill, c.dir, c.fill)
		}
	}
}
