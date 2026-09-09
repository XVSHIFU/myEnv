package state

import (
	"fmt"
	"golang.org/x/sys/unix"
)

func processIdentity(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid process ID")
	}
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("darwin:%d:%d", info.Proc.P_starttime.Sec, info.Proc.P_starttime.Usec), nil
}
