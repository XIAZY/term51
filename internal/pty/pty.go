// Package pty opens a pseudo-terminal and starts a program on it.
//
// The master side stays in this process; the program gets the slave side as
// its standard input, output and error and as its controlling terminal, in a
// new session. The pty is opened with the POSIX calls posix_openpt, grantpt,
// unlockpt and ptsname, which QNX (BlackBerry 10) and Linux both provide, so
// the same code builds for the phone and for tests on the development
// machine. creack/pty has no QNX support, which is why this package exists.
package pty

/*
// glibc declares posix_openpt, grantpt, unlockpt and ptsname only with these
// feature macros; QNX declares them unconditionally.
#ifdef __linux__
#define _GNU_SOURCE
#endif
#include <fcntl.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <termios.h>
#include <sys/ioctl.h>

// open_master opens a pty master and prepares its slave. On success it
// returns the master descriptor and copies the slave's path into name; on
// failure it returns -1 with errno set.
static int open_master(char *name, size_t len) {
	int fd = posix_openpt(O_RDWR | O_NOCTTY);
	if (fd < 0) {
		return -1;
	}
	if (grantpt(fd) != 0 || unlockpt(fd) != 0) {
		int e = errno;
		close(fd);
		errno = e;
		return -1;
	}
	char *p = ptsname(fd);
	if (p == NULL || strlen(p) >= len) {
		int e = (p == NULL) ? errno : ENAMETOOLONG;
		close(fd);
		errno = e;
		return -1;
	}
	strcpy(name, p);
	return fd;
}

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

// Open opens a new pseudo-terminal. It returns the master side, which this
// process reads the program's output from and writes input to, and the slave
// side, which is given to the program.
func Open() (master, slave *os.File, err error) {
	var name [256]C.char
	fd, cerr := C.open_master(&name[0], C.size_t(len(name)))
	if fd < 0 {
		return nil, nil, fmt.Errorf("pty: opening the master: %w", cerr)
	}
	master = os.NewFile(uintptr(fd), "/dev/ptmx")
	path := C.GoString(&name[0])
	slave, err = os.OpenFile(path, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: opening the slave %s: %w", path, err)
	}
	return master, slave, nil
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
// controlling terminal; cmd's Stdin, Stdout, Stderr and SysProcAttr must be
// unset. The caller reads and writes the master, and closes it after cmd has
// been waited for.
func Start(cmd *exec.Cmd, size Size) (*os.File, error) {
	if cmd.Stdin != nil || cmd.Stdout != nil || cmd.Stderr != nil || cmd.SysProcAttr != nil {
		return nil, errors.New("pty: cmd must not set Stdin, Stdout, Stderr or SysProcAttr")
	}
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
