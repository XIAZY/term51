//go:build qnx

package pty

/*
#include <devctl.h>
#include <errno.h>
#include <string.h>
#include <sys/dcmd_chr.h>
#include <sys/procfs.h>

// readers fills pids with up to max processes blocked reading the terminal
// on fd and returns how many are blocked, or -1 with the error in *err.
static int readers(int fd, int *pids, int max, int *err) {
	char buf[sizeof(struct _ttywaitinfo) + 16 * sizeof(struct _pidtid)];
	struct _ttywaitinfo *w = (struct _ttywaitinfo *)buf;
	memset(buf, 0, sizeof buf);
	w->queue = TTY_READ_Q;
	w->num = 16;
	int r = devctl(fd, DCMD_CHR_WAITINFO, buf, sizeof buf, NULL);
	if (r != EOK) {
		*err = r;
		return -1;
	}
	int n = w->num < (unsigned)max ? (int)w->num : max;
	for (int i = 0; i < n; i++)
		pids[i] = w->blocked[i].pid;
	return (int)w->num;
}

// process reads what procfs says of the process whose as file is open on fd.
static int process(int fd, int *pgrp, int *child, int *sibling, unsigned long long *start) {
	procfs_info info;
	int r = devctl(fd, DCMD_PROC_INFO, &info, sizeof info, NULL);
	if (r != EOK)
		return r;
	*pgrp = info.pgrp;
	*child = info.child;
	*sibling = info.sibling;
	*start = info.start_time;
	return EOK;
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Foreground finds the process in the foreground of the terminal whose
// master is master and whose session leader (the shell Start started) is
// leader. The phone's devc-pty answers tcgetpgrp only to a process whose
// controlling terminal it is, which this one is not; what it does answer,
// on the slave side, is which processes are blocked reading the terminal.
// So: a process reading the terminal (vi, pi, the shell at its prompt) is
// the one; when none reads (a build running), the newest process group
// among the leader's descendants other than its own; and the leader when
// there is none. A background job can be taken for the foreground only
// while nothing reads the terminal.
//
// The slave is opened only for the question: held open, it would keep the
// master from seeing the shell's end.
func Foreground(master *os.File, leader int) (int, error) {
	name := master.Name()
	slavePath := filepath.Join(filepath.Dir(name), "tty"+strings.TrimPrefix(filepath.Base(name), "pty"))
	slave, err := os.OpenFile(slavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return 0, fmt.Errorf("pty: %w", err)
	}
	var pids [16]C.int
	var n C.int
	var cerr C.int
	conn, err := slave.SyscallConn()
	if err == nil {
		err = conn.Control(func(fd uintptr) { n = C.readers(C.int(fd), &pids[0], 16, &cerr) })
	}
	slave.Close()
	if err != nil {
		return 0, fmt.Errorf("pty: %w", err)
	}
	if n < 0 {
		return 0, fmt.Errorf("pty: who reads %s: %w", slavePath, syscall.Errno(cerr))
	}
	if n > 0 {
		return int(pids[0]), nil
	}
	return newestGroup(leader)
}

// newestGroup is the leader of the newest process group among leader's
// descendants other than leader's own, or leader when there is none.
func newestGroup(leader int) (int, error) {
	best, bestStart := leader, uint64(0)
	todo := []int{leader}
	for len(todo) > 0 {
		pid := todo[0]
		todo = todo[1:]
		p, err := procInfo(pid)
		if err != nil {
			continue // it has ended since its parent was read
		}
		if pid != leader && p.pgrp != leader && p.pgrp == pid && p.start > bestStart {
			best, bestStart = pid, p.start
		}
		// procfs links a process to its first child and each child to the
		// next; the leader's own siblings are not its descendants.
		if p.child > 0 {
			todo = append(todo, p.child)
		}
		if pid != leader && p.sibling > 0 {
			todo = append(todo, p.sibling)
		}
	}
	return best, nil
}

type procfsInfo struct {
	pgrp, child, sibling int
	start                uint64
}

func procInfo(pid int) (procfsInfo, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/as", pid))
	if err != nil {
		return procfsInfo{}, err
	}
	defer f.Close()
	var pgrp, child, sibling C.int
	var start C.ulonglong
	var r C.int
	conn, err := f.SyscallConn()
	if err != nil {
		return procfsInfo{}, err
	}
	if err := conn.Control(func(fd uintptr) { r = C.process(C.int(fd), &pgrp, &child, &sibling, &start) }); err != nil {
		return procfsInfo{}, err
	}
	if r != 0 {
		return procfsInfo{}, syscall.Errno(r)
	}
	return procfsInfo{pgrp: int(pgrp), child: int(child), sibling: int(sibling), start: uint64(start)}, nil
}

// ProgramPath is the path the process pid was started by.
func ProgramPath(pid int) (string, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/exefile", pid))
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(b), "\x00 \n"), nil
}
