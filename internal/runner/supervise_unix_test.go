//go:build !windows

package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestWaitDescendantWithoutInheritedPipes(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Unlike pipe-retaining descendants, this child cannot accidentally keep
	// exec.Cmd.Wait alive through stdout/stderr copy goroutines.
	code, err := Execute(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "(sleep 0.3; printf done > finished) </dev/null >/dev/null 2>&1 & exit 0"}, Directory: root, Environment: os.Environ()})
	if err != nil || code != 0 {
		t.Fatalf("parent result %d %v", code, err)
	}
	_, evidenceErr := os.Stat(filepath.Join(root, "finished"))
	// Allow the short test-owned descendant to finish even on the current
	// failing implementation, before its temporary directory is removed.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(root, "finished")); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if evidenceErr != nil {
		t.Fatal("runner returned while its descendant was still using the generation", evidenceErr)
	}
}

func TestSignalBufferedBeforeChildStart(t *testing.T) {
	observed := make(chan os.Signal, 1)
	signal.Notify(observed, syscall.SIGTERM)
	defer signal.Stop(observed)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", "exec sleep 20")
	supervisor, err := prepareSupervision(ctx, command, "")
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	if err = syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-observed:
	case <-ctx.Done():
		t.Fatal("signal was not delivered")
	}
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	if err = supervisor.Start(); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal(err)
	}
	_ = command.Wait()
	if err = supervisor.Finish(); !errors.Is(err, ErrTreeUnconfirmed) {
		t.Fatalf("group signaling claimed complete tree supervision: %v", err)
	}
	if code := processExitCode(command.ProcessState); code != 128+int(syscall.SIGTERM) {
		t.Fatalf("queued signal lost: code %d", code)
	}
}

func TestDirectUnixCompletionUnconfirmed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	code, err := executeDirect(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "exit 17"}})
	if code != 1 || !errors.Is(err, ErrTreeUnconfirmed) {
		t.Fatalf("direct fallback released tree protection: %d %v", code, err)
	}
}

func TestCancelProcessGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	var output bytes.Buffer
	started := time.Now()
	code, err := Execute(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "sleep 30 & wait"}, Directory: t.TempDir(), Environment: os.Environ(), Stdout: &output, Stderr: &output})
	if err != nil || code != 128+9 {
		t.Fatalf("code=%d error=%v", code, err)
	}
	// The sleep child inherits the output pipe. Killing only the shell would
	// keep Wait blocked until that pipe closes after thirty seconds.
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("descendant retained output pipe for %s", elapsed)
	}
}

func TestCancelProcessGroupAfterParentExit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	var output bytes.Buffer
	started := time.Now()
	// The shell exits immediately, while sleep keeps the output pipe open.
	code, err := Execute(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "sleep 3 & exit 0"}, Directory: t.TempDir(), Environment: os.Environ(), Stdout: &output, Stderr: &output})
	if err != nil || code == 0 {
		t.Fatalf("cancelled process group: code=%d error=%v", code, err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("cancellation stopped after parent exit: %s", elapsed)
	}
}
