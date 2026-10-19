//go:build !qnx

package pty

/*
#include <unistd.h>
*/
import "C"

import (
	"fmt"
	"os"
	"strings"
)

// Foreground is the process group in the foreground of the terminal whose
// master is master (tcgetpgrp), or leader, the shell Start started, while
// none is (tcgetpgrp answers 0).
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
	if pgrp == 0 {
		return leader, nil
	}
	return int(pgrp), nil
}

// ProgramPath is the path the process pid was started by, its first
// argument, as on QNX (a link's own name, not what it points to).
func ProgramPath(pid int) (string, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	arg0, _, _ := strings.Cut(string(b), "\x00")
	return arg0, nil
}
