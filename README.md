# term51

A terminal emulator for BlackBerry 10, written in Go with a Cascades UI.
The terminal core is `github.com/XIAZY/libghostty-vt-go`, a port of
libghostty-vt (Ghostty v1.3.1).

## Layout (planned)

- `main.go`: keeps the main goroutine on the process's main thread for the
  Qt event loop and starts the app.
- `internal/ui`: the Cascades bridge (one C++ `QObject`) and the native window
  the terminal is drawn into (`ForeignWindowControl`).
- `internal/pty`: the pseudo-terminal and the shell.
- `internal/session`: one goroutine per terminal; feeds the shell's output to
  the library and key input back to the shell.
- `internal/render`: draws the library's render state into the window.
- `internal/replies`: the app's own answers to programs' queries.
- `assets/*.qml`: the pages.

## Workflow

`main` is the development line. Every change is made on its own feature
branch and merged into `main` only after review: the reviewer reads the diff
and reruns the tests.
