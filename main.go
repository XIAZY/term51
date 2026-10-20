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
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	_ "golang.org/x/crypto/x509roots/fallback" // root certificates: the phone has none where Go looks
	"golang.org/x/image/font/opentype"

	"term51/internal/app"
	"term51/internal/busybox"
	"term51/internal/cells"
	"term51/internal/gpu"
	"term51/internal/layout"
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

	// The app's folder is where it is started: app/native holds the
	// package, data is its home, logs the sandbox's folder for its logs.
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("the app's folder: %v", err)
	}
	places := layout.For(root)
	log.SetFlags(log.Lmicroseconds)
	if f, err := os.OpenFile(places.Log, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	} else {
		log.Printf("the log goes to stderr only: %v", err)
	}
	if err := places.Make(); err != nil {
		log.Printf("the app's folders: %v", err)
	}

	// The settings: a text file in the app's home. A file with a mistake
	// in it is not used at all, and the log says what is wrong. The first
	// start writes the file with the defaults (settings.Load).
	settingsFile := places.Config
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
	a := app.New(cfg, ui.Publisher{}, func(group, id string, w, h int, es int) (app.Screen, error) {
		return gpu.Open(group, id, w, h, es)
	}, phoneFonts(), func(s settings.Settings) error { return settings.Save(settingsFile, s) })
	a.Bundled = bundled(root, places)
	// PowerShell is not in the package: the settings sheet downloads it
	// into the app's own data (.local/share/term51), with the certificates of the Go team's copy of
	// Mozilla's list (the fallback import), since the phone keeps its own
	// where Go does not look.
	// pwsh is linked into the user's own programs, at the front of every
	// shell's PATH, so that it is found by name from any tab (pi-go's
	// powershell tool looks for it so).
	var link func(pwsh.State)
	store := pwsh.Open(places.Share, pwsh.ManifestURL, pwsh.Target, &http.Client{}, func(st pwsh.State) {
		link(st)
		a.PowerShellChanged(st)
	})
	a.Bundled.PowerShell = store.Program()
	a.PowerShell = store
	link = func(st pwsh.State) {
		if a.Bundled.Bin == "" {
			return
		}
		if err := pwsh.SyncLink(filepath.Join(a.Bundled.Bin, "pwsh"), a.Bundled.PowerShell, st); err != nil {
			log.Printf("linking pwsh: %v", err)
		}
	}
	link(store.State())
	// kill -USR1 reads the next frame back into the data folder, for
	// comparing what OpenGL ES 3.0 and 2.0 draw.
	usr1 := make(chan os.Signal, 1)
	signal.Notify(usr1, syscall.SIGUSR1)
	readBack := make(chan struct{})
	go func() {
		for range usr1 {
			readBack <- struct{}{}
		}
	}()
	a.ReadBack, a.ReadBackDir = readBack, places.Home
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
// they are. The package is read-only; what is written goes in the app's
// home. root is the app's folder.
//
// BusyBox's tools are run by their names (head, tr, ash ...) through links
// to it in places.Tools, which goes at the end of the shells' PATH; the
// Dropbear SSH client's names (ssh, scp ...) are links there as well. The links
// are checked at every start and made again when the package brought
// another BusyBox. Without them the app still runs: the log says why, and
// the shells get no such folder.
//
// The app's home (its data folder) gets the conventional .local/bin, at the
// front of the shells' PATH; the tmp folder of the app's sandbox, which the
// system makes, is their TMPDIR.
func bundled(root string, places layout.Places) settings.Bundled {
	var b settings.Bundled
	tools := places.Tools
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
	// The BusyBox shell's start-up file, whose prompt reports the
	// directory that names the tab.
	if env := filepath.Join(root, "app/native/busybox/ashrc"); isFile(env) {
		b.AshEnv = env
	} else {
		log.Printf("bundled: the package has no %s; the BusyBox shell's tabs show no directory", env)
	}
	// The user's own programs (where pwsh is linked), in the conventional
	// place in the app's home, its data folder.
	b.Bin = settings.HomeBin(os.Getenv("HOME"))
	if b.Bin == "" {
		log.Printf("bundled: no HOME; the shells get no .local/bin, and pwsh is not linked")
	}
	if tmp := filepath.Join(root, "tmp"); isDir(tmp) {
		b.Tmp = tmp
	} else {
		log.Printf("bundled: the sandbox has no %s; the shells get no TMPDIR of the app's", tmp)
	}
	if b.Bin != "" {
		if err := os.MkdirAll(b.Bin, 0o700); err != nil {
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

// isDir reports whether path is a directory.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}
