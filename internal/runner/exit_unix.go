//go:build linux || darwin

package runner

import (
	"os"
	"syscall"
)

func processExitCode(state *os.ProcessState) int {
	if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return state.ExitCode()
}
