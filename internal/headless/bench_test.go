package headless

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The whole pipeline under heavy output: a program writes a file to the
// terminal as fast as it can, the session parses it, and the view draws a
// frame whenever the terminal reports a change, into alternating buffers.
// The result is the throughput of the app as a whole, drawing included.
//
//	go test ./internal/headless -run '^$' -bench . -benchtime 3x

func flood(b *testing.B, line string) {
	// Not under /tmp, which is a RAM disk on the development machine.
	dir, err := os.MkdirTemp(".", "flood")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
	var data bytes.Buffer
	for data.Len() < 4<<20 {
		data.WriteString(line)
	}
	file, err := filepath.Abs(filepath.Join(dir, "out.txt"))
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(file, data.Bytes(), 0o600); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(data.Len()))
	frames, rows := 0, 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, err := Open(Options{
			Shell: "/bin/bash", Args: []string{"--norc", "--noprofile", "-c", `cat "$1"; printf '\n@@END@@\n'; sleep 30`, "bash", file},
			Env: []string{"PATH=/usr/bin:/bin"},
			W:   1440, H: 1440, FontPx: 32, // the Passport's screen
		})
		if err != nil {
			b.Fatal(err)
		}
		if err := w.WaitText("@@END@@", 5*time.Minute); err != nil {
			w.Close()
			b.Fatal(err)
		}
		f, r := w.Stats()
		frames += f
		rows += r
		b.StopTimer()
		w.Close()
		b.StartTimer()
	}
	b.ReportMetric(float64(frames)/float64(b.N), "frames/op")
	b.ReportMetric(float64(rows)/float64(b.N)/1440, "screens/op")
}

func BenchmarkFloodPlain(b *testing.B) {
	flood(b, "The quick brown fox jumps over the lazy dog 0123456789 abcdefghij\r\n")
}

func BenchmarkFloodColour(b *testing.B) {
	flood(b, "\x1b[1;32mhello world \x1b[0m some plain text here \x1b[38;5;208m1234567890\x1b[0m \x1b[48;2;10;20;30mrgb\x1b[0m\r\n")
}

// One typed character echoed back and drawn: the round trip a user feels.
func BenchmarkKeyEcho(b *testing.B) {
	w, err := Open(Options{
		Shell: "/bin/bash", Args: []string{"--norc", "--noprofile", "-c", `stty -icanon min 1 2>/dev/null; cat`},
		Env: []string{"PATH=/usr/bin:/bin"},
		W:   1440, H: 1440, FontPx: 32,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer w.Close()
	if err := w.Settle(100*time.Millisecond, 2*time.Second); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%60 == 59 {
			if err := w.Type("\n"); err != nil {
				b.Fatal(err)
			}
		}
		before, _ := w.Stats()
		if err := w.Type("x"); err != nil {
			b.Fatal(err)
		}
		// Until the echo has been drawn.
		deadline := time.Now().Add(5 * time.Second)
		for {
			select {
			case <-w.Session.Changed():
			case <-time.After(time.Millisecond):
			}
			if err := w.Frame(); err != nil {
				b.Fatal(err)
			}
			if after, _ := w.Stats(); after > before {
				break
			}
			if time.Now().After(deadline) {
				b.Fatal("the echo was never drawn")
			}
		}
	}
}
