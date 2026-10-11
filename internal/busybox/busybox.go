// Package busybox sets up the tools of the BusyBox the app ships: one
// program that is all of them, each by its own name through a symbolic
// link. The package holds the program; the links are made in the app's
// data folder, which is the only place the app can write.
package busybox

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// stamp is the file in the links' folder that says which BusyBox they were
// made for.
const stamp = ".busybox"

// Links makes sure dir holds a link for every tool of the BusyBox at
// program (head, tr, ash ...), by "busybox --install -s". They are made
// when the folder is not there, and made again when the program is another
// one than they were made for: another file, or the same file at another
// place, which is what the links point to. Both paths must be absolute.
func Links(program, dir string) error {
	sum, err := checksum(program)
	if err != nil {
		return fmt.Errorf("busybox: %w", err)
	}
	want := program + "\n" + sum + "\n"
	if have, err := os.ReadFile(filepath.Join(dir, stamp)); err == nil && string(have) == want {
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("busybox: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("busybox: %w", err)
	}
	if out, err := exec.Command(program, "--install", "-s", dir).CombinedOutput(); err != nil {
		return fmt.Errorf("busybox: making the links in %s: %v: %s", dir, err, out)
	}
	// Written last: links that were only half made are made again.
	if err := os.WriteFile(filepath.Join(dir, stamp), []byte(want), 0o644); err != nil {
		return fmt.Errorf("busybox: %w", err)
	}
	return nil
}

func checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
