//go:build qnx

// term51 is a terminal emulator for BlackBerry 10, written in Go with a
// Cascades UI. The terminal core is libghostty-vt-go; the only code that is
// not Go is the Cascades bridge (internal/ui).
package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"term51/internal/app"
	"term51/internal/gpu"
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
	// The first start writes the file with the defaults (settings.Load).
	cfg, warnings, err := settings.Load("data/settings.conf")
	for _, w := range warnings {
		log.Printf("settings: %s", w)
	}
	if err != nil {
		log.Printf("settings: %v; using the defaults", err)
	}
	// The process ends in one place only: here on the main thread, after
	// the UI's event loop has returned and Cascades has taken itself down.
	// On this system os.Exit is the C library's exit, which runs the
	// libraries' clean-up on the thread that calls it. Called from another
	// thread, with the UI still up or just going down, it left a thread
	// waiting for the phone's input service for ever and a process that
	// could not be killed (found on the phone, 2026-10-07).
	a := app.New(cfg, ui.Publisher{}, func(group, id string, w, h int) (app.Screen, error) {
		return gpu.Open(group, id, w, h)
	})
	ended := make(chan struct{})
	go func() {
		err := a.Run()
		log.Printf("the app's loop ended: %v", err)
		// The last shell is gone: the app has nothing left to show. End
		// the UI; main does the rest. This comes before ended is closed,
		// so that main cannot exit while this goroutine is still inside
		// the C++ call.
		ui.Quit()
		close(ended)
	}()

	rc := ui.Run(func(name, argsJSON string) {
		var args map[string]any
		if argsJSON != "" {
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				log.Printf("ui: bad arguments for %s: %v", name, err)
				return
			}
		}
		a.Send(app.Command{Name: name, Args: args})
	})
	// The UI has ended, because the user closed the app or the last shell
	// went. Hang up any shells still running, with a bounded wait, then
	// leave.
	a.Quit()
	select {
	case <-ended:
	case <-time.After(4 * time.Second):
	}
	os.Exit(rc)
}
