package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestJobControlFailureReapsTree(t *testing.T) {
	if mode := os.Getenv("MYENV_CONTROL_ERROR_HELPER"); mode != "" {
		failure := errors.New("injected terminal control failure")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		forward := make(chan os.Signal, 1)
		command := exec.Command("/bin/sh", "-c", "sleep 30 & kill -STOP $$; wait")
		command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
		stopped := func() error {
			if mode == "stop-cancel" {
				cancel()
				return failure
			}
			if mode == "stop" {
				return failure
			}
			forward <- syscall.SIGCONT
			return nil
		}
		status, complete, err := reapDescendantsJobControl(ctx, func() (int, error) {
			if err := command.Start(); err != nil {
				return 0, err
			}
			return command.Process.Pid, nil
		}, forward, stopped, func() error { return failure })
		if command.Process != nil {
			defer command.Process.Release()
		}
		if !complete || !errors.Is(err, failure) || !status.Signaled() || status.Signal() != unix.SIGKILL {
			t.Fatalf("control error cleanup: complete=%v status=%v err=%v", complete, status, err)
		}
		if mode == "stop-cancel" && !errors.Is(err, context.Canceled) {
			t.Fatalf("concurrent cancellation lost: %v", err)
		}
		var remaining unix.WaitStatus
		if _, err := unix.Wait4(-1, &remaining, unix.WNOHANG, nil); err != unix.ECHILD {
			t.Fatalf("owned descendants remain: %v", err)
		}
		return
	}
	for _, mode := range []string{"stop", "continue", "stop-cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestJobControlFailureReapsTree$")
			command.Env = append(os.Environ(), "MYENV_CONTROL_ERROR_HELPER="+mode)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("dedicated reaper: %v\n%s", err, output)
			}
		})
	}
}
