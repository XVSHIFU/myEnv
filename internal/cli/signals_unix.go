//go:build !windows

package cli

import (
	"os"
	"syscall"
)

func operationSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}
