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

	"term51/internal/app"
	"term51/internal/settings"
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

	// The settings: a text file in the app's data directory. A file with a
	// mistake in it is not used at all, and the log says what is wrong.
	cfg, warnings, err := settings.Load("data/settings.conf")
	for _, w := range warnings {
		log.Printf("settings: %s", w)
	}
	if err != nil {
		log.Printf("settings: %v; using the defaults", err)
	}
	a := app.New(cfg, ui.Publisher{})
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
