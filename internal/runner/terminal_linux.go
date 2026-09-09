package runner

import (
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// Only the dedicated supervisor calls this after launch, on stop or completion.
// Ignoring SIGTTOU here cannot change a subsequently executed user's disposition:
// this process launches no further children.
func restoreTerminalForeground(command *exec.Cmd, previous int) error {
	current, err := unix.IoctlGetInt(0, unix.TIOCGPGRP)
	if err != nil {
		return err
	}
	if current == previous {
		return nil
	}
	if command.Process != nil && current != command.Process.Pid {
		// A shell may already have reclaimed the terminal after caller death.
		return nil
	}
	// Start can fail after the fork has set foreground but before exec, leaving
	// command.Process nil. The saved caller group still needs to be restored.
	signal.Ignore(syscall.SIGTTOU)
	return unix.IoctlSetPointerInt(0, unix.TIOCSPGRP, previous)
}

func resumeTerminalForeground(command *exec.Cmd, caller int) error {
	current, err := unix.IoctlGetInt(0, unix.TIOCGPGRP)
	if err != nil {
		return err
	}
	// bg must not steal the terminal from the shell or another foreground job.
	if current != caller || command.Process == nil {
		return nil
	}
	signal.Ignore(syscall.SIGTTOU)
	return unix.IoctlSetPointerInt(0, unix.TIOCSPGRP, command.Process.Pid)
}
