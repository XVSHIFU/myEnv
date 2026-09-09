package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDedicatedSubreaperDetachedDescendant(t *testing.T) {
	testDedicatedSubreaper(t, false)
}

func TestDedicatedSubreaperCancellation(t *testing.T) { testDedicatedSubreaper(t, true) }

func testDedicatedSubreaper(t *testing.T, canceled bool) {
	if root := os.Getenv("MYENV_SUBREAPER_TEST_ROOT"); root != "" {
		script := "setsid /bin/sh -c 'sleep 0.3; printf done > finished' </dev/null >/dev/null 2>&1 & exit 17"
		ctx := context.Background()
		if canceled {
			script = "setsid /bin/sh -c 'printf started > started; sleep 20' </dev/null >/dev/null 2>&1 & exit 17"
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, 300*time.Millisecond)
			defer cancel()
		}
		command := exec.Command("/bin/sh", "-c", script)
		command.Dir = root
		command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
		status, complete, err := reapDescendantsContext(ctx, func() (int, error) {
			if err := command.Start(); err != nil {
				return 0, err
			}
			return command.Process.Pid, nil
		})
		if command.Process != nil {
			_ = command.Process.Release()
		}
		if !complete {
			t.Fatal("tree completion unproven", err)
		}
		if canceled {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("cancellation missing", err)
			}
			if _, err := os.Stat(filepath.Join(root, "started")); err != nil {
				t.Fatal("detached child never started", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if contents, err := os.ReadFile(filepath.Join(root, "finished")); !canceled && (err != nil || string(contents) != "done") {
			t.Fatal("subreaper returned early", err)
		}
		if !status.Exited() {
			t.Fatal("leader did not exit normally")
		}
		os.Exit(status.ExitStatus())
	}
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Fatal("test requires setsid", err)
	}
	root := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^"+t.Name()+"$")
	command.Env = append(os.Environ(), "MYENV_SUBREAPER_TEST_ROOT="+root)
	output, err := command.CombinedOutput()
	if command.ProcessState == nil || command.ProcessState.ExitCode() != 17 {
		t.Fatalf("lost leader exit or descendant: %v %s", err, output)
	}
}
