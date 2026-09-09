package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestPreparationChildReceipt(t *testing.T) {
	for _, retain := range []bool{false, true} {
		name := "normal"
		if retain {
			name = "retain_after_completion"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			unlock, err := state.LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
			if err != nil {
				t.Fatal(err)
			}
			defer unlock()
			s, err := state.Open(ctx, filepath.Join(root, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.BeginGuardedOperation(ctx, "operation", filepath.Join(root, "generation"), "digest"); err != nil {
				t.Fatal(err)
			}
			child, err := s.BeginOperationChild(ctx, "operation")
			if err != nil {
				t.Fatal(err)
			}
			p := runner.Process{Executable: "/bin/sh", Args: []string{"-c", "exit 7"}, Environment: os.Environ(), TreeID: child}
			closeReceipt, err := prepareChildCompletion(ctx, root, s, "operation", child, &p)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if closeReceipt != nil {
					_ = closeReceipt(false)
				}
			}()
			code, err := runner.Execute(ctx, p)
			if err != nil || code != 7 {
				t.Fatalf("child: %d %v", code, err)
			}
			path := filepath.Join(root, child+".complete")
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "myenv-tree-complete-v1:"+p.Completion.Token+"\n" {
				t.Fatalf("receipt: %q %v", data, err)
			}
			if !retain {
				if err := s.CompleteOperationChild(ctx, "operation", child); err != nil {
					t.Fatal(err)
				}
			}
			if err := closeReceipt(!retain); err != nil {
				t.Fatal(err)
			}
			closeReceipt = nil
			_, err = os.Stat(path)
			if (retain && err != nil) || (!retain && !os.IsNotExist(err)) {
				t.Fatalf("receipt retention: %v", err)
			}
			if err := s.ConfirmOperationTreesDone(ctx, "operation"); (err != nil) != retain {
				t.Fatalf("pending child protection: %v", err)
			}
		})
	}
}
