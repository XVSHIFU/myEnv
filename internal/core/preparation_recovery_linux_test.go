package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestPreparationRecoveryExitedOwner(t *testing.T) {
	ctx := context.Background()
	if root := os.Getenv("MYENV_PREPARATION_RECOVERY_HELPER"); root != "" {
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
		for i := 1; i <= 6; i++ {
			id := fmt.Sprintf("%032x", i)
			dir := filepath.Join(work, "generations", id)
			opdir := filepath.Join(work, "operations", id)
			for _, path := range []string{dir, opdir} {
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "partial"), []byte("fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := s.BeginGuardedOperation(ctx, id, dir, "digest"); err != nil {
				t.Fatal(err)
			}
			child, err := s.BeginOperationChild(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			p := runner.Process{Executable: "/bin/sh", Args: []string{"-c", "exit 0"}, Environment: os.Environ(), TreeID: child}
			closeReceipt, err := prepareChildCompletion(ctx, opdir, s, id, child, &p)
			if err != nil {
				t.Fatal(err)
			}
			code, runErr := runner.Execute(ctx, p)
			closeErr := closeReceipt(false)
			if runErr != nil || closeErr != nil || code != 0 {
				t.Fatalf("child %d %v %v", code, runErr, closeErr)
			}
			if i == 2 {
				if err := os.WriteFile(filepath.Join(opdir, child+".complete"), []byte("wrong token"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if i == 3 {
				// Register a second child but leave its receipt empty, as when
				// the owner exits before launching the already registered child.
				pending, err := s.BeginOperationChild(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				var pendingProcess runner.Process
				closePending, err := prepareChildCompletion(ctx, opdir, s, id, pending, &pendingProcess)
				if err != nil {
					t.Fatal(err)
				}
				if err := closePending(false); err != nil {
					t.Fatal(err)
				}
			}
			if i >= 4 {
				path := filepath.Join(opdir, child+".complete")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				switch i {
				case 4:
					// Even a symlink to the exact valid token must be rejected.
					if err := os.WriteFile(path+".original", data, 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(filepath.Base(path)+".original", path); err != nil {
						t.Fatal(err)
					}
				case 5:
					if err := unix.Mkfifo(path, 0600); err != nil {
						t.Fatal(err)
					}
				case 6:
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		return // Exit with receipts, deliberately without child DB completion.
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
	command := exec.Command(executable, "-test.run=^TestPreparationRecoveryExitedOwner$")
	command.Env = append(os.Environ(), "MYENV_PREPARATION_RECOVERY_HELPER="+root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("owner: %v %s", err, output)
	}
	service := &Service{}
	database := filepath.Join(root, ".myenv", "state.db")
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.Clean(ctx, root, true, func(item CleanItem) error {
		if item.ID != fmt.Sprintf("%032x", 1) || item.Removed {
			t.Fatalf("preview included protected operation: %+v", item)
		}
		return nil
	})
	if err != nil || preview.Candidates != 1 || preview.Changed || preview.Removed != 0 || preview.RecoveredPreparations != 0 {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	after, err := os.ReadFile(database)
	if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatalf("preview changed database: %v", err)
	}
	result, err := service.Clean(ctx, root, false, nil)
	if err != nil || result.RecoveredPreparations != 1 || result.Removed != 1 {
		t.Fatalf("clean: %+v %v", result, err)
	}
	if result.Candidates != preview.Candidates || result.Bytes != preview.Bytes {
		t.Fatalf("preview/actual mismatch: %+v %+v", preview, result)
	}
	for i := 1; i <= 6; i++ {
		_, err := os.Stat(filepath.Join(root, ".myenv", "generations", fmt.Sprintf("%032x", i), "partial"))
		if (i == 1 && !os.IsNotExist(err)) || (i != 1 && err != nil) {
			t.Fatalf("generation %d: %v", i, err)
		}
	}
	store, err := state.OpenReadOnly(ctx, filepath.Join(root, ".myenv", "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	children, err := store.PreparationChildren(ctx, fmt.Sprintf("%032x", 3), "")
	closeErr := store.Close()
	completed := 0
	for _, child := range children {
		if child.Completed {
			completed++
		}
	}
	if err != nil || closeErr != nil || len(children) != 2 || completed != 1 {
		t.Fatalf("partial recovery children=%+v error=%v close=%v", children, err, closeErr)
	}
	result, err = service.Clean(ctx, root, false, nil)
	if err != nil || result.Changed {
		t.Fatalf("repeat clean: %+v %v", result, err)
	}
}
