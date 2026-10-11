//go:build qnx

// term51 is a terminal emulator for BlackBerry 10, written in Go with a
// Cascades UI. The terminal core is libghostty-vt-go; the only code that is
// not Go is the Cascades bridge (internal/ui).
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	_ "golang.org/x/crypto/x509roots/fallback" // root certificates: the phone has none where Go looks
	"golang.org/x/image/font/opentype"

	"term51/internal/app"
	"term51/internal/busybox"
	"term51/internal/cells"
	"term51/internal/gpu"
	"term51/internal/pwsh"
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
	const settingsFile = "data/settings.conf"
	cfg, warnings, err := settings.Load(settingsFile)
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
	}, phoneFonts(), func(s settings.Settings) error { return settings.Save(settingsFile, s) })
	// The app's folder is where it is started: app/native holds the
	// package, data is its own to write.
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("the app's folder: %v", err)
	}
	a.Bundled = bundled(root)
	// PowerShell is not in the package: the settings sheet downloads it
	// into the data folder, with the certificates of the Go team's copy of
	// Mozilla's list (the fallback import), since the phone keeps its own
	// where Go does not look.
	store := pwsh.Open(filepath.Join(root, "data"), pwsh.ManifestURL, pwsh.Target, &http.Client{}, a.PowerShellChanged)
	a.Bundled.PowerShell = store.Program()
	a.PowerShell = store
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

// bundled sets up the programs that come in the package, and says where
// they and PowerShell's folders are. The package is read-only; what is
// written goes in the data folder. root is the app's folder.
//
// BusyBox's tools are run by their names (head, tr, ash ...) through links
// to it in data/bin, which goes at the end of the shells' PATH; the
// Dropbear SSH client's names (ssh, scp ...) are links there as well. The links
// are checked at every start and made again when the package brought
// another BusyBox. Without them the app still runs: the log says why, and
// the shells get no such folder.
//
// PowerShell is given a home folder and a folder for temporary files.
func bundled(root string) settings.Bundled {
	b := settings.Bundled{
		PowerShellHome: filepath.Join(root, "data/pwsh-home"),
		PowerShellTmp:  filepath.Join(root, "data/pwsh-tmp"),
	}
	tools := filepath.Join(root, "data/bin")
	// The Dropbear SSH client is one program too: ssh and the rest by name.
	dropbear := busybox.Tool{
		Program: filepath.Join(root, "app/native/dropbear/dropbearmulti"),
		Names:   []string{"ssh", "dbclient", "scp", "dropbearkey", "dropbearconvert"},
	}
	if err := busybox.Links(filepath.Join(root, "app/native/busybox/busybox"), tools, dropbear); err != nil {
		log.Printf("%v; the shells get no BusyBox tools and no ssh", err)
	} else {
		b.Tools = tools
	}
	for _, dir := range []string{b.PowerShellHome, b.PowerShellTmp} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			log.Printf("bundled: %v", err)
		}
	}
	return b
}

// The phone's own font files; the app ships none.
const fontDir = "/usr/fonts/font_repository/"

// phoneFonts reads the fonts the terminal is drawn with: DejaVu Sans Mono in
// its four styles, and for the characters it lacks MHei (Chinese, and the
// characters Japanese shares with it), then Heisei Kaku Gothic (the Japanese
// kana, which MHei lacks) and then Malgun (Korean). The files are part of
// the phone, so one that cannot be read ends the app, with the reason in the
// log.
func phoneFonts() cells.Fonts {
	read := func(name string) *opentype.Font {
		data, err := os.ReadFile(fontDir + name)
		if err != nil {
			log.Fatalf("fonts: %v", err)
		}
		f, err := opentype.Parse(data)
		if err != nil {
			log.Fatalf("fonts: %s: %v", name, err)
		}
		return f
	}
	return cells.Fonts{
		Faces: [4]*opentype.Font{
			read("dejavu/DejaVuSansMono.ttf"),
			read("dejavu/DejaVuSansMono-Bold.ttf"),
			read("dejavu/DejaVuSansMono-Oblique.ttf"),
			read("dejavu/DejaVuSansMono-BoldOblique.ttf"),
		},
		Others: []*opentype.Font{
			read("monotype/MHeiM.GB18030_200910.ttf"),
			read("monotype/HeiseiKakuGothW5_213_C.ttf"),
			read("monotype/malgun.ttf"),
		},
	}
}
