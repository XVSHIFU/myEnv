package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRecoverInterruptedKeepsActive(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	release, err := LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	g := Generation{ID: "active", Directory: filepath.Join(root, "active"), InputDigest: "d", NodeExecutable: filepath.Join(root, "active", "node.exe")}
	if err = s.Publish(ctx, g, ""); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOperation(ctx, "interrupted", filepath.Join(root, "pending"), "d"); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginGuardedOperation(ctx, "settled", filepath.Join(root, "settled"), "d"); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfirmOperationTreesDone(ctx, "settled"); err != nil {
		t.Fatal(err)
	}
	count, err := s.RecoverInterrupted(ctx)
	if err != nil || count != 1 {
		t.Fatalf("recovery count %d %v", count, err)
	}
	var status string
	if err = s.db.QueryRow(`SELECT status FROM operations WHERE id='interrupted'`).Scan(&status); err != nil || status != "preparing" {
		t.Fatalf("recovered legacy operation without tree proof: %s %v", status, err)
	}
	active, err := s.Active(ctx)
	if err != nil || active == nil || active.ID != g.ID {
		t.Fatal("recovery changed active generation")
	}
	count, err = s.RecoverInterrupted(ctx)
	if err != nil || count != 0 {
		t.Fatal("recovery not idempotent")
	}
}
