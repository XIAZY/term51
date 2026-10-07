//go:build qnx

package pty

import "os"

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
