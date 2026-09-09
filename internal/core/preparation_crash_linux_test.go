package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestPreparationRecoveryKilledOwner(t *testing.T) {
	const id = "00000000000000000000000000000099"
	ctx := context.Background()
	if root := os.Getenv("MYENV_PREPARATION_CRASH_HELPER"); root != "" {
		work := filepath.Join(root, ".myenv")
		unlock, err := state.LockWorkspace(ctx, filepath.Join(work, "modify.lock"))
		if err != nil {
			t.Fatal(err)
		}
		defer unlock()
		s, err := state.Open(ctx, filepath.Join(work, "state.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		dir, opdir := filepath.Join(work, "generations", id), filepath.Join(work, "operations", id)
		for _, p := range []string{dir, opdir} {
			if err := os.MkdirAll(p, 0700); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "partial"), []byte("protected"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := s.BeginGuardedOperation(ctx, id, dir, "digest"); err != nil {
			t.Fatal(err)
		}
		child, err := s.BeginOperationChild(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		p := runner.Process{Executable: "/bin/sh", Args: []string{"-c", "sleep 30 & printf ready > started; wait"}, Directory: root, Environment: os.Environ(), TreeID: child}
		closeReceipt, err := prepareChildCompletion(ctx, opdir, s, id, child, &p)
		if err != nil {
			t.Fatal(err)
		}
		defer closeReceipt(false)
		_, err = runner.Execute(ctx, p)
		t.Fatalf("owner unexpectedly survived: %v", err)
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".myenv"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestPreparationRecoveryKilledOwner$")
	command.Env = append(os.Environ(), "MYENV_PREPARATION_CRASH_HELPER="+root)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child never started")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// While the owner holds the modification lock, actual cleanup must wait.
	blocked, cancel := context.WithTimeout(ctx, 80*time.Millisecond)
	result, err := (&Service{}).Clean(blocked, root, false, nil)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) || result.Changed {
		t.Fatalf("clean bypassed live owner: %+v %v", result, err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("owner was not killed")
	}
	waited = true
	deadline = time.Now().Add(5 * time.Second)
	for {
		result, err = (&Service{}).Clean(ctx, root, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		if result.RecoveredPreparations == 1 {
			if result.Removed != 1 {
				t.Fatalf("recovered without cleanup: %+v", result)
			}
			break
		}
		if result.Removed != 0 {
			t.Fatalf("deleted without completion: %+v", result)
		}
		if time.Now().After(deadline) {
			t.Fatal("independent completion receipt never recovered")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv", "generations", id)); !os.IsNotExist(err) {
		t.Fatalf("generation still exists: %v", err)
	}
}
