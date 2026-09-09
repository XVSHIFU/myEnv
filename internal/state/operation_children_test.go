package state

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationChildCompletion(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	unlock, err := LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	// Tokens cannot be attached to another child or overwritten after binding.
	defer unlock()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.BeginGuardedOperation(ctx, "pending", filepath.Join(root, "generation"), "digest"); err != nil {
		t.Fatal(err)
	}
	child, err := s.BeginOperationChild(ctx, "pending")
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("ab", 32)
	if err := s.BindOperationChildCompletion(ctx, "pending", child, "bad"); err == nil {
		t.Fatal("accepted invalid token")
	}
	if err := s.BindOperationChildCompletion(ctx, "pending", "wrong", token); err == nil {
		t.Fatal("accepted wrong child binding")
	}
	if err := s.BindOperationChildCompletion(ctx, "pending", child, token); err != nil {
		t.Fatal(err)
	}
	if err := s.BindOperationChildCompletion(ctx, "pending", child, token); err == nil {
		t.Fatal("replaced child binding")
	}
	if err := s.ConfirmOperationTreesDone(ctx, "pending"); err == nil {
		t.Fatal("confirmed outstanding child")
	}
	if err := s.FailGuardedOperation(ctx, "pending"); err == nil {
		t.Fatal("released outstanding child")
	}
	if err := s.CompleteOperationChild(ctx, "pending", "wrong"); err == nil {
		t.Fatal("accepted wrong child")
	}
	if err := s.CompleteOperationChild(ctx, "pending", child); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteOperationChild(ctx, "pending", child); err == nil {
		t.Fatal("accepted duplicate completion")
	}
	if err := s.ConfirmOperationTreesDone(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BeginOperationChild(ctx, "pending"); err == nil {
		t.Fatal("launched after closing child phase")
	}
	if err := s.FailGuardedOperation(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
}
