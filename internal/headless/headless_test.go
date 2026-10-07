package headless

import (
	"testing"
	"time"

	"term51/internal/keys"
)

func open(t *testing.T, script string) *Window {
	t.Helper()
	w, err := Open(Options{
		Shell: "/bin/bash", Args: []string{"--norc", "--noprofile", "-c", script},
		Env: []string{"PATH=/usr/bin:/bin", "PS1=$ "},
		W:   640, H: 367, FontPx: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	return w
}

// lit counts the pixels of the shown buffer that differ from the background.
// The last pixel is background: the test window's height is not a whole
// number of rows, so the bottom edge is outside the grid. (The first pixel
// is not: the cursor starts there.)
func lit(w *Window) int {
	pix := w.Pixels()
	bg := pix[len(pix)-1]
	n := 0
	for _, p := range pix {
		if p != bg {
			n++
		}
	}
	return n
}

// The whole way through: typed keys reach the shell, its output reaches the
// screen, and the screen reaches the pixels.
func TestKeysInPixelsOut(t *testing.T) {
	w := open(t, `read -r line; echo "got:$line"; sleep 5`)
	before := lit(w)
	if err := w.Type("Hello\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.WaitText("got:Hello", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(50*time.Millisecond, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if after := lit(w); after <= before {
		t.Errorf("pixels lit: %d before typing, %d after", before, after)
	}
	if w.Image() == nil {
		t.Error("no image")
	}
}

// The key bar's sticky Ctrl, then a letter, arrives as the control
// character: Ctrl+D ends cat's input.
func TestStickyCtrlReachesTheProgram(t *testing.T) {
	w := open(t, `cat; echo "cat ended"; sleep 5`)
	w.Keys.StickyCtrl(true)
	if err := w.Type("d"); err != nil {
		t.Fatal(err)
	}
	if err := w.WaitText("cat ended", 10*time.Second); err != nil {
		t.Fatal(err)
	}
}

// Frames drawn row by row into alternating buffers end up identical to one
// full redraw, after output that scrolls and rewrites lines.
func TestAlternatingBuffersMatchAFullRedraw(t *testing.T) {
	w := open(t, `for i in $(seq 1 60); do printf 'line %d \033[1;3%dmcolour\033[0m\n' $i $((i%7+1)); done; printf 'done\n'; sleep 5`)
	if err := w.WaitText("done", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	// Both buffers have to see the final state.
	for i := 0; i < 3; i++ {
		if err := w.Settle(30*time.Millisecond, time.Second); err != nil {
			t.Fatal(err)
		}
	}
	got := append([]uint32(nil), w.Pixels()...)

	w.View.Invalidate()
	if err := w.Frame(); err != nil {
		t.Fatal(err)
	}
	want := w.Pixels()
	diff := 0
	for i := range want {
		if got[i] != want[i] {
			diff++
		}
	}
	if diff != 0 {
		t.Errorf("%d of %d pixels differ from a full redraw", diff, len(want))
	}
	if frames, rows := w.Stats(); frames < 2 || rows == 0 {
		t.Errorf("stats: %d frames, %d pixel rows", frames, rows)
	}
}

// An arrow key in a program that switched the cursor keys to application
// mode is sent in that mode's form.
func TestArrowKeyFollowsTheProgramsMode(t *testing.T) {
	w := open(t, `printf '\033[?1h'; IFS= read -r -s -n 3 k; printf 'key:%q\n' "$k"; sleep 5`)
	if err := w.Settle(100*time.Millisecond, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := w.Key(keys.Raw{Down: true, Sym: keys.CodeUp, Cap: keys.CodeUp}); err != nil {
		t.Fatal(err)
	}
	if err := w.WaitText(`key:$'\EOA'`, 10*time.Second); err != nil {
		t.Fatal(err)
	}
}
