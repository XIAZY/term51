// term51 is a terminal emulator for BlackBerry 10, written in Go with a
// Cascades UI. The terminal core is libghostty-vt-go; C++ is limited to the
// Cascades bridge and C to the native window (internal/ui).
//
// This is milestone 0: a proof that Go draws into a native window inside the
// Cascades page and receives key presses (internal/m0).
package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"

	"term51/internal/m0"
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
		defer f.Close()
	}
	log.SetFlags(log.Lmicroseconds)

	a := m0.New()
	go a.Run()

	rc := ui.Run(func(name, argsJSON string) {
		var args map[string]any
		if argsJSON != "" {
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				log.Printf("ui: bad arguments for %s: %v", name, err)
				return
			}
		}
		a.Send(m0.Command{Name: name, Args: args})
	})
	os.Exit(rc)
}
