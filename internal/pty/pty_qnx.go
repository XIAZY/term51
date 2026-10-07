//go:build qnx

package pty

import (
	"os"
	"syscall"
)

// Open opens a new pseudo-terminal: the first free BSD-style device pair in
// /dev. It returns the master side, which this process reads the program's
// output from and writes input to, and the slave side, which is given to the
// program.
//
// BlackBerry 10's headers declare posix_openpt, grantpt, unlockpt and
// ptsname, but its C library does not contain them, and there is no
// /dev/ptmx.
func Open() (master, slave *os.File, err error) {
	return openBSD(bsdDevDir)
}

// Go raises the limit on open files when a program starts and remembers the
// old one to give back to the programs it starts; giving it back is one of
// the things that makes it start them with vfork instead of spawn (see
// Start). Setting the limit ourselves, to what it now is, tells Go there is
// nothing to give back. The shell then starts with the raised limit.
func init() {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err == nil {
		syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim)
	}
}
