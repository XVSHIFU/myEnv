package cli

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

func openPromptTerminal(file *os.File) (int, error) {
	return unix.Open(fmt.Sprintf("/proc/self/fd/%d", file.Fd()), unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOCTTY, 0)
}
