package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestTerminalGroupResumesGrandchild(t *testing.T) {
	if os.Getenv("MYENV_GROUP_RESUME_HELPER") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminalGroupResumesGrandchild$")
		cmd.Env = append(os.Environ(), "MYENV_GROUP_RESUME_HELPER=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	forward := make(chan os.Signal, 1)
	command := exec.Command("/bin/sh", "-c", "sleep 0.1 & kill -STOP -$$; wait; exit 23")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	status, complete, err := reapDescendantsJobControl(ctx, func() (int, error) {
		if err := command.Start(); err != nil {
			return 0, err
		}
		return command.Process.Pid, nil
	}, forward, func() error { forward <- syscall.SIGCONT; return nil }, func() error { return nil })
	if err != nil || !complete || status.ExitStatus() != 23 {
		t.Fatal(fmt.Sprintf("status=%v complete=%v err=%v", status, complete, err))
	}
}
