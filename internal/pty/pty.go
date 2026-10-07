// Package pty opens a pseudo-terminal and starts a program on it.
//
// The master side stays in this process; the program gets the slave side as
// its standard input, output and error and as its controlling terminal, in a
// new session. How the pty is opened depends on the system, one way each: on
// BlackBerry 10, whose C library has no posix_openpt, the BSD devices
// /dev/pty[p-s]? (pty_qnx.go); elsewhere, where the tests run, the POSIX
// calls (pty_posix.go). creack/pty has no QNX support, which is why this
// package exists.
package pty

/*
#include <string.h>
#include <termios.h>
#include <sys/ioctl.h>

static int set_size(int fd, unsigned short rows, unsigned short cols,
		unsigned short xpixel, unsigned short ypixel) {
	struct winsize ws;
	memset(&ws, 0, sizeof ws);
	ws.ws_row = rows;
	ws.ws_col = cols;
	ws.ws_xpixel = xpixel;
	ws.ws_ypixel = ypixel;
	return ioctl(fd, TIOCSWINSZ, &ws);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Size is a terminal size in cells, with the size in pixels for programs
// that ask for it (zero when unknown).
type Size struct {
	Rows, Cols     uint16
	XPixel, YPixel uint16
}

// bsdDevDir is where BSD-style pseudo-terminals live; tests change it.
var bsdDevDir = "/dev"

// BSD pseudo-terminal names: masters dir/ptyXY, slaves dir/ttyXY, with X in
// bsdBanks and Y in bsdUnits.
const (
	bsdBanks = "pqrs"
	bsdUnits = "0123456789abcdefghijklmnopqrstuv"
)

// openBSD opens the first free BSD-style pseudo-terminal in dir: a master
// dir/ptyXY that opens (one already in use refuses a second open) and its
// slave dir/ttyXY.
func openBSD(dir string) (master, slave *os.File, err error) {
	tried := 0
	for _, x := range bsdBanks {
		for _, y := range bsdUnits {
			mpath := fmt.Sprintf("%s/pty%c%c", dir, x, y)
			m, err := os.OpenFile(mpath, os.O_RDWR|syscall.O_NOCTTY, 0)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				tried++
				continue // in use, or not permitted
			}
			tried++
			spath := fmt.Sprintf("%s/tty%c%c", dir, x, y)
			s, err := os.OpenFile(spath, os.O_RDWR|syscall.O_NOCTTY, 0)
			if err != nil {
				m.Close()
				continue
			}
			return m, s, nil
		}
	}
	if tried == 0 {
		return nil, nil, fmt.Errorf("no BSD pseudo-terminals in %s", dir)
	}
	return nil, nil, fmt.Errorf("all %d BSD pseudo-terminals in %s are in use or refused", tried, dir)
}

// SetSize sets the terminal size the program sees (TIOCSWINSZ). The kernel
// sends SIGWINCH to the program's foreground process group.
func SetSize(f *os.File, s Size) error {
	conn, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var serr error
	cerr := conn.Control(func(fd uintptr) {
		_, serr = C.set_size(C.int(fd), C.ushort(s.Rows), C.ushort(s.Cols),
			C.ushort(s.XPixel), C.ushort(s.YPixel))
	})
	if cerr != nil {
		return cerr
	}
	if serr != nil {
		return fmt.Errorf("pty: setting the size: %w", serr)
	}
	return nil
}

// Start starts cmd on a new pseudo-terminal of the given size and returns
// the master side. The program runs in a new session with the pty as its
// controlling terminal; Start sets cmd's Stdin, Stdout, Stderr and
// SysProcAttr. The caller reads and writes the master, and closes it after cmd has
// been waited for.
func Start(cmd *exec.Cmd, size Size) (*os.File, error) {
	master, slave, err := Open()
	if err != nil {
		return nil, err
	}
	// The slave is only needed until the child has it.
	defer slave.Close()
	if err := SetSize(master, size); err != nil {
		master.Close()
		return nil, err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	// Ctty is a descriptor number in the child: 0 is the slave as stdin.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := cmd.Start(); err != nil {
		master.Close()
		return nil, fmt.Errorf("pty: starting %s: %w", cmd.Path, err)
	}
	return master, nil
}
