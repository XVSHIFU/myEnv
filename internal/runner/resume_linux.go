package runner

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// The caller closes ack only after leaving its stop/read loop. Repeating CONT
// until that acknowledgment also covers a STOP issued after the first CONT.
// The inherited pidfd identifies the original caller even after PID reuse.
func resumeCallerUntilAcknowledged(pidfd, ack int) error {
	fds := []unix.PollFd{{Fd: int32(ack), Events: unix.POLLIN}}
	for {
		if _, err := unix.Poll(fds, 0); err != nil && err != unix.EINTR {
			return err
		}
		if fds[0].Revents&(unix.POLLERR|unix.POLLNVAL) != 0 {
			return fmt.Errorf("invalid caller acknowledgment pipe")
		}
		if fds[0].Revents&(unix.POLLIN|unix.POLLHUP) != 0 {
			var b [1]byte
			n, err := unix.Read(ack, b[:])
			if err == unix.EINTR {
				continue
			}
			if err != nil {
				return err
			}
			if n != 0 {
				return fmt.Errorf("unexpected caller acknowledgment data")
			}
			return nil
		}
		if err := unix.PidfdSendSignal(pidfd, unix.SIGCONT, nil, 0); err != nil {
			if err == unix.ESRCH {
				return nil
			}
			return err
		}
		if _, err := unix.Poll(fds, 10); err != nil && err != unix.EINTR {
			return err
		}
	}
}
