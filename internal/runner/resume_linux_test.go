package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestResumeCallerAcknowledgment(t *testing.T) {
	for _, mode := range []string{"closed", "data", "data-closed", "invalid-pidfd"} {
		t.Run(mode, func(t *testing.T) {
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer read.Close()
			defer write.Close()
			if strings.HasPrefix(mode, "data") {
				if _, err := write.Write([]byte("x")); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "closed" || mode == "data-closed" {
				write.Close()
			}
			// No valid process handle: an accepted EOF must return without signaling.
			err = resumeCallerUntilAcknowledged(-1, int(read.Fd()))
			switch mode {
			case "closed":
				if err != nil {
					t.Fatal(err)
				}
			case "invalid-pidfd":
				if !errors.Is(err, unix.EBADF) {
					t.Fatalf("signal error lost: %v", err)
				}
			default:
				if err == nil || !strings.Contains(err.Error(), "unexpected caller acknowledgment data") {
					t.Fatalf("data accepted as acknowledgment: %v", err)
				}
			}
		})
	}
}

func TestResumeCallerDelayedStop(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	defer write.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", "sleep 0.1; kill -STOP $$; exec 3>&-")
	command.ExtraFiles = []*os.File{write}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
	write.Close()
	fd, err := unix.PidfdOpen(command.Process.Pid, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if err := resumeCallerUntilAcknowledged(fd, int(read.Fd())); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("late stop did not resume: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("deadline rescued hung resume protocol")
	}
}
