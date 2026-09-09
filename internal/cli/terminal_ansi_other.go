//go:build !windows

package cli

import "os"

func enableTerminalANSI(f *os.File) bool { return true }
