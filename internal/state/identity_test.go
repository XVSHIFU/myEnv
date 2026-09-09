package state

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestProcessIdentityLifecycle(t *testing.T) {
	if os.Getenv("MYENV_IDENTITY_CHILD") == "1" {
		identity, err := supervisorIdentity()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(identity)
		for {
			time.Sleep(time.Second)
		}
	}
	for _, pid := range []int{0, -1} {
		if _, err := processIdentity(pid); err == nil || errors.Is(err, ErrProcessGone) {
			t.Fatal("invalid PID treated as observed process")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessIdentityLifecycle$")
	cmd.Env = append(os.Environ(), "MYENV_IDENTITY_CHILD=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatal("child did not report startup identity")
	}
	reported := scanner.Text()
	for i := 0; i < 2; i++ {
		identity, err := processIdentity(cmd.Process.Pid)
		if err != nil || identity != reported {
			t.Fatalf("external identity %q, self %q: %v", identity, reported, err)
		}
	}
	if err = cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	if err == nil {
		t.Fatal("expected child termination")
	}
	_, err = processIdentity(cmd.Process.Pid)
	if err == nil {
		t.Fatal("returned identity after confirmed child exit")
	}
	// Darwin's sized sysctl may return EIO for an absent process; keep that
	// observation unknown until a native absence contract is implemented.
	if runtime.GOOS != "darwin" && !errors.Is(err, ErrProcessGone) {
		t.Fatalf("missing death evidence: %v", err)
	}
}
