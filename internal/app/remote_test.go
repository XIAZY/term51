package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"term51/internal/settings"
)

// The development aid: commands written to a file are run in order, the
// page's reach the page, and the file is emptied.
func TestRemoteCommands(t *testing.T) {
	file := filepath.Join(t.TempDir(), "remote")
	a, p, done := startAppWith(t, func(cfg *settings.Settings, a *App) {
		cfg.Remote = true
		a.RemoteFile = file
	})
	if err := os.WriteFile(file, []byte("type echo from-afar\\n\ndrawer open\nwait 50\nshot one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wait(t, "the typed command's output", func() bool { return strings.Count(p.screen(), "from-afar") == 2 })
	wait(t, "the page's commands", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return reflect.DeepEqual(p.remote, []string{"drawer open", "shot one"})
	})
	if data, err := os.ReadFile(file); err != nil || len(data) != 0 {
		t.Errorf("the command file afterwards: %q, %v", data, err)
	}
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
