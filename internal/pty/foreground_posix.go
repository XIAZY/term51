//go:build !qnx

package pty

/*
#include <unistd.h>
*/
import "C"

import (
	"fmt"
	"os"
)

// Foreground is the process group in the foreground of the terminal whose
// master is master (tcgetpgrp); leader is not needed here (see the QNX
// version for why it is there).
func Foreground(master *os.File, leader int) (int, error) {
	conn, err := master.SyscallConn()
	if err != nil {
		return 0, err
	}
	var pgrp C.pid_t
	var cerr error
	if err := conn.Control(func(fd uintptr) { pgrp, cerr = C.tcgetpgrp(C.int(fd)) }); err != nil {
		return 0, err
	}
	if pgrp < 0 {
		return 0, fmt.Errorf("pty: tcgetpgrp: %w", cerr)
	}
	return int(pgrp), nil
}

// ProgramPath is the program the process pid runs.
func ProgramPath(pid int) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
}
