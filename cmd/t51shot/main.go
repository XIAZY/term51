// t51shot runs a program in the app's terminal without a window and writes
// what the screen would show as a PNG. It goes through everything the app
// does on the phone except the window itself: the session, the key
// translation, and the view drawing frames into alternating buffers.
//
//	t51shot -o shot.png -type 'ls --color\n' -wait '$ ' -- /bin/bash --norc
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"strings"
	"time"

	"term51/internal/headless"
)

func main() {
	out := flag.String("o", "t51shot.png", "the PNG to write")
	width := flag.Int("w", 1440, "width of the drawing area in pixels")
	height := flag.Int("h", 1440, "height of the drawing area in pixels")
	font := flag.Float64("font", 32, "font size in pixels")
	typed := flag.String("type", "", `text to type once the program has started (\n is Enter, \t is Tab)`)
	wait := flag.String("wait", "", "text to wait for on the screen before the picture is taken")
	quiet := flag.Duration("quiet", 300*time.Millisecond, "how long the screen must be still before the picture is taken")
	limit := flag.Duration("limit", 30*time.Second, "give up after this long")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: t51shot [flags] -- program [arguments]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*out, *width, *height, *font, *typed, *wait, *quiet, *limit, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "t51shot:", err)
		os.Exit(1)
	}
}

func run(out string, w, h int, font float64, typed, wait string, quiet, limit time.Duration, cmd []string) error {
	win, err := headless.Open(headless.Options{Shell: cmd[0], Args: cmd[1:], W: w, H: h, FontPx: font})
	if err != nil {
		return err
	}
	defer win.Close()
	if err := win.Settle(quiet, limit); err != nil {
		return err
	}
	if typed != "" {
		typed = strings.NewReplacer(`\n`, "\n", `\t`, "\t").Replace(typed)
		if err := win.Type(typed); err != nil {
			return err
		}
	}
	if wait != "" {
		if err := win.WaitText(wait, limit); err != nil {
			return err
		}
	}
	if err := win.Settle(quiet, limit); err != nil {
		return err
	}
	img := win.Image()
	if img == nil {
		return fmt.Errorf("nothing was drawn")
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	frames, rows := win.Stats()
	fmt.Printf("%s: %dx%d, %d frames, %d pixel rows painted\n", out, w, h, frames, rows)
	return f.Close()
}
