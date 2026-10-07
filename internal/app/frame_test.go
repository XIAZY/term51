package app

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// The picture read back from the screen has its bottom row first; the file
// has the top row first.
func TestSaveFrameTurnsThePictureTheRightWayUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	// 2 by 2: the bottom row is red, green; the top row is blue, white.
	shot := []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 255}
	if err := saveFrame(path, shot, 2, 2); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		x, y    int
		r, g, b uint32
	}{{0, 0, 0, 0, 255}, {1, 0, 255, 255, 255}, {0, 1, 255, 0, 0}, {1, 1, 0, 255, 0}} {
		r, g, b, _ := img.At(c.x, c.y).RGBA()
		if r>>8 != c.r || g>>8 != c.g || b>>8 != c.b {
			t.Errorf("pixel (%d,%d) is (%d,%d,%d), want (%d,%d,%d)", c.x, c.y, r>>8, g>>8, b>>8, c.r, c.g, c.b)
		}
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("%d files left in the directory", len(entries))
	}
}
