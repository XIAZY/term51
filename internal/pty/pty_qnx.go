//go:build qnx

package pty

import (
	"errors"
	"os"
)

// openPOSIX has nothing to call on BlackBerry 10: the headers declare
// posix_openpt, grantpt, unlockpt and ptsname, but the C library does not
// contain them (the link fails). Open falls back to the BSD devices.
func openPOSIX() (master, slave *os.File, err error) {
	return nil, nil, errors.New("posix_openpt: not in this system's C library")
}
