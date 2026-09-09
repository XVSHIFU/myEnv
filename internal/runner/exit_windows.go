//go:build windows

package runner

import "os"

func processExitCode(state *os.ProcessState) int { return state.ExitCode() }
