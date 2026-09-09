package core

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestPreparationRecoveryKilledWindowsOwner(t *testing.T) {
	const id = "00000000000000000000000000000099"
	ctx := context.Background()
	if os.Getenv("MYENV_PREPARATION_WINDOWS_WORKER") == "1" {
		root := os.Getenv("MYENV_PREPARATION_WINDOWS_CRASH_HELPER")
		if err := os.WriteFile(filepath.Join(root, "started"), []byte("ready"), 0600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Second)
		return
	}
	if root := os.Getenv("MYENV_PREPARATION_WINDOWS_CRASH_HELPER"); root != "" {
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
		if _, err := s.BeginOperationChild(ctx, id); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "tree-id"), []byte(child), 0600); err != nil {
			t.Fatal(err)
		}
		p := runner.Process{Executable: os.Args[0], Args: []string{"-test.run=^TestPreparationRecoveryKilledWindowsOwner$"}, Directory: root, Environment: append(os.Environ(), "MYENV_PREPARATION_WINDOWS_WORKER=1"), TreeID: child}
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
	command := exec.Command(executable, "-test.run=^TestPreparationRecoveryKilledWindowsOwner$")
	command.Env = append(os.Environ(), "MYENV_PREPARATION_WINDOWS_CRASH_HELPER="+root)
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
	treeID, err := os.ReadFile(filepath.Join(root, "tree-id"))
	if err != nil {
		t.Fatal(err)
	}
	jobName, err := windows.UTF16PtrFromString(`Local\myEnv-lease-` + string(treeID))
	if err != nil {
		t.Fatal(err)
	}
	jobValue, _, openErr := windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenJobObjectW").Call(4, 0, uintptr(unsafe.Pointer(jobName)))
	if jobValue == 0 {
		t.Fatalf("retain fixture Job handle: %v", openErr)
	}
	job := windows.Handle(jobValue)
	defer func() {
		if job != 0 {
			windows.CloseHandle(job)
		}
	}()
	live, err := (&Service{}).Clean(ctx, root, true, nil)
	if err != nil || live.Candidates != 0 || live.Changed {
		t.Fatalf("preview included live owner: %+v %v", live, err)
	}
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
	protected, err := (&Service{}).Clean(ctx, root, true, nil)
	if err != nil || protected.Candidates != 0 || protected.Changed {
		t.Fatalf("preview ignored retained Job: %+v %v", protected, err)
	}
	partial, err := (&Service{}).Clean(ctx, root, false, nil)
	if err != nil || !partial.Changed || partial.RecoveredPreparations != 0 || partial.Removed != 0 {
		t.Fatalf("partial Job recovery: %+v %v", partial, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv", "generations", id, "partial")); err != nil {
		t.Fatalf("deleted active Job generation: %v", err)
	}
	if err := windows.CloseHandle(job); err != nil {
		t.Fatal(err)
	}
	job = 0 // Last Job handle closes; KILL_ON_JOB_CLOSE ends the fixture worker.
	database := filepath.Join(root, ".myenv", "state.db")
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	var preview CleanResult
	for {
		preview, err = (&Service{}).Clean(ctx, root, true, nil)
		if err != nil {
			t.Fatal(err)
		}
		if preview.Changed || preview.Removed != 0 || preview.RecoveredPreparations != 0 {
			t.Fatalf("preview mutated state: %+v", preview)
		}
		if preview.Candidates == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Windows Job disappearance never recovered")
		}
		time.Sleep(20 * time.Millisecond)
	}
	after, err := os.ReadFile(database)
	if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatalf("preview changed database bytes: %v", err)
	}
	result, err = (&Service{}).Clean(ctx, root, false, nil)
	if err != nil || result.RecoveredPreparations != 1 || result.Removed != 1 || result.Candidates != preview.Candidates || result.Bytes != preview.Bytes {
		t.Fatalf("preview/actual mismatch: %+v %+v %v", preview, result, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv", "generations", id)); !os.IsNotExist(err) {
		t.Fatalf("generation still exists: %v", err)
	}
}
