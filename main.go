//go:build qnx

// term51 is a terminal emulator for BlackBerry 10, written in Go with a
// Cascades UI. The terminal core is libghostty-vt-go; C++ is limited to the
// Cascades bridge and C to the native window (internal/ui).
package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"term51/internal/app"
	"term51/internal/ui"
)

// The UI runs on the main goroutine, kept on the process's main thread.
func init() { runtime.LockOSThread() }

func main() {
	// The phone has about 2 GB shared by everything; without a limit the Go
	// heap can grow until the system kills the app.
	debug.SetMemoryLimit(512 << 20)

	// Apps run in their sandbox, whose data directory is writable.
	if f, err := os.OpenFile("data/term51.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	}
	log.SetFlags(log.Lmicroseconds)

	cfg := app.DefaultConfig()
	// Which front end: the file data/frontend holds "text" or "pixels"
	// (until there is a settings page).
	if b, err := os.ReadFile("data/frontend"); err == nil {
		if f := strings.TrimSpace(string(b)); f == app.FrontendText || f == app.FrontendPixels {
			cfg.Frontend = f
		}
	}
	// data/rawkeys, when it exists, makes the pixel front end read raw
	// Screen key events instead of Cascades key events.
	if _, err := os.Stat("data/rawkeys"); err == nil {
		cfg.RawKeys = true
	}
	a := app.New(cfg)
	go func() {
		err := a.Run()
		log.Printf("the session ended: %v", err)
		// The shell is gone: the app has nothing left to show.
		os.Exit(0)
	}()

	rc := ui.Run(a.Page(), func(name, argsJSON string) {
		var args map[string]any
		if argsJSON != "" {
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				log.Printf("ui: bad arguments for %s: %v", name, err)
				return
			}
		}
		a.Send(app.Command{Name: name, Args: args})
	})
	os.Exit(rc)
}
