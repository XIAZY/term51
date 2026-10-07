// Package pty opens a pseudo-terminal and starts a program on it.
//
// The master side stays in this process; the program gets the slave side as
// its standard input, output and error and as its controlling terminal, in a
// new session, and no other descriptor of this process (see Start). How the
// pty is opened depends on the system, one way each: on
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

// set_ctty makes the terminal on fd the controlling terminal of the calling
// process, which must be a session leader without one.
static int set_ctty(int fd) {
	return ioctl(fd, TIOCSCTTY, 0);
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

// childArg is the first argument of this program when it is started as the
// step between Start and the program to run (see child). It is not something
// a user would type.
const childArg = "--term51-pty-child--"

// init runs the child step when this program was started for it. It is in
// init, not in main, so that it comes before anything else the program
// does, and so that every program that imports this package, the tests
// included, starts shells the same way.
func init() {
	if len(os.Args) >= 4 && os.Args[1] == childArg {
		child(os.Args[2], os.Args[3], os.Args[4:])
	}
}

// child is the step between Start and the program: it runs in the new
// process, which Start made a session leader with the terminal as its
// descriptors 0, 1 and 2 and nothing else open. It makes the terminal the
// controlling terminal, changes to dir if one is given, and becomes the
// program. It does not return.
func child(dir, path string, argv []string) {
	fail := func(what string, err error) {
		// The terminal is the only place to say it.
		fmt.Fprintf(os.Stderr, "term51: %s: %v\r\n", what, err)
		os.Exit(127)
	}
	if r, err := C.set_ctty(0); r != 0 {
		fail("making the terminal the controlling terminal", err)
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			fail("changing to "+dir, err)
		}
	}
	fail("starting "+path, syscall.Exec(path, argv, os.Environ()))
}

// Start starts cmd on a new pseudo-terminal of the given size, in a new
// session with the terminal as its controlling terminal and as its standard
// input, output and error. It returns the master side; the caller reads and
// writes it, and closes it after cmd has been waited for. Start rewrites
// cmd's Path, Args, Dir, Stdin, Stdout, Stderr and SysProcAttr.
//
// The program is not started directly. Start starts this same executable
// with childArg, as a session leader holding only the terminal, and that
// process then becomes the program (child). The reason is which descriptors
// the program gets. Go can start a process two ways on BlackBerry 10: with
// spawn, which hands over exactly the descriptors listed, or with vfork,
// which copies every descriptor this process has open. Asking Go to set the
// controlling terminal forces vfork, and then the shell holds copies of the
// connections Cascades opened in the app, to the phone's input service
// among others. Setting the controlling terminal in the child step instead
// leaves Go free to use spawn. (Elsewhere, where the tests run, Go has no
// spawn and the program gets what any child gets; the tests check the
// session, the terminal, the directory and the arguments, and the
// descriptors are checked on the phone.)
func Start(cmd *exec.Cmd, size Size) (*os.File, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("pty: %w", err)
	}
	// A program that is not there (a mistyped shell in the settings) is
	// the caller's to report. Once the child step has started, the
	// terminal is the only place left for an error.
	if _, err := os.Stat(cmd.Path); err != nil {
		return nil, fmt.Errorf("pty: starting %s: %w", cmd.Path, err)
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
	program := cmd.Path
	cmd.Args = append([]string{self, childArg, cmd.Dir, program}, cmd.Args...)
	cmd.Path, cmd.Dir = self, ""
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		master.Close()
		return nil, fmt.Errorf("pty: starting %s: %w", program, err)
	}
	return master, nil
}
