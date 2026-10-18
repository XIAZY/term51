package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFor(t *testing.T) {
	p := For("/app")
	want := Places{
		Home:   "/app/data",
		Config: "/app/data/.config/term51/settings.conf",
		Log:    "/app/logs/term51.log",
		Share:  "/app/data/.local/share/term51",
		Tools:  "/app/data/.local/share/term51/bin",
	}
	if p != want {
		t.Errorf("For(/app) = %+v, want %+v", p, want)
	}
}

func TestMake(t *testing.T) {
	p := For(t.TempDir())
	if err := p.Make(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Dir(p.Config), p.Share} {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			t.Errorf("%s: %v", dir, err)
		}
	}
}
