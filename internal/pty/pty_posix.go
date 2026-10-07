//go:build !qnx

package pty

/*
// glibc declares posix_openpt, grantpt, unlockpt and ptsname only with this
// feature macro.
#ifdef __linux__
#define _GNU_SOURCE
#endif
#include <fcntl.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>

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
*/
import "C"

import (
	"fmt"
	"os"
	"syscall"
)

// openPOSIX opens a pseudo-terminal with posix_openpt, grantpt, unlockpt and
// ptsname.
func openPOSIX() (master, slave *os.File, err error) {
	var name [256]C.char
	fd, cerr := C.open_master(&name[0], C.size_t(len(name)))
	if fd < 0 {
		return nil, nil, fmt.Errorf("posix_openpt: %w", cerr)
	}
	master = os.NewFile(uintptr(fd), "/dev/ptmx")
	path := C.GoString(&name[0])
	slave, err = os.OpenFile(path, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("opening the slave %s: %w", path, err)
	}
	return master, slave, nil
}
