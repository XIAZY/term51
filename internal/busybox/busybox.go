// Package busybox sets up the tools of the BusyBox the app ships: one
// program that is all of them, each by its own name through a symbolic
// link. The package holds the program; the links are made in the app's
// data folder, which is the only place the app can write. Other programs
// of the package that go by several names (the Dropbear SSH client) get
// their links in the same folder.
package busybox

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// stamp is the file in the links' folder that says which programs they
// were made for.
const stamp = ".busybox"

// Tool is another program of the package, and the names it is run by.
type Tool struct {
	Program string
	Names   []string
}

// Links makes sure dir holds a link for every tool of the BusyBox at
// program (head, tr, ash ...), by "busybox --install -s", and for every
// name of the other tools. They are made when the folder is not there, and
// made again when a program is another one than they were made for:
// another file, or the same file at another place, which is what the links
// point to. All paths must be absolute.
func Links(program, dir string, tools ...Tool) error {
	sum, err := checksum(program)
	if err != nil {
		return fmt.Errorf("busybox: %w", err)
	}
	want := program + "\n" + sum + "\n"
	for _, t := range tools {
		sum, err := checksum(t.Program)
		if err != nil {
			return fmt.Errorf("busybox: %w", err)
		}
		want += t.Program + "\n" + sum + "\n" + strings.Join(t.Names, " ") + "\n"
	}
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
	for _, t := range tools {
		for _, name := range t.Names {
			link := filepath.Join(dir, name)
			// A name BusyBox has too goes to the other tool.
			if err := os.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("busybox: %w", err)
			}
			if err := os.Symlink(t.Program, link); err != nil {
				return fmt.Errorf("busybox: %w", err)
			}
		}
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
