package state

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the database CAS separately from platform owner-death observation.
func TestPreparationRecoveryObservedIdentity(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	unlock, err := LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.BeginGuardedOperation(ctx, "pending", filepath.Join(root, "generation"), "digest"); err != nil {
		t.Fatal(err)
	}
	id, err := s.BeginOperationChild(ctx, "pending")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.BindOperationChildCompletion(ctx, "pending", id, strings.Repeat("ab", 32)); err != nil {
		t.Fatal(err)
	}
	owners, err := s.PreparingChildOperations(ctx, "")
	if err != nil || len(owners) != 1 {
		t.Fatal(owners, err)
	}
	children, err := s.PreparationChildren(ctx, "pending", "")
	if err != nil || len(children) != 1 {
		t.Fatal(children, err)
	}
	owner, child := owners[0], children[0]
	for _, field := range []string{"operation", "pid", "identity", "child", "token"} {
		o, c := owner, child
		switch field {
		case "operation":
			o.ID = "other"
		case "pid":
			o.PID = -1
		case "identity":
			o.Identity = "stale"
		case "child":
			c.ID = "other"
		case "token":
			c.Token = strings.Repeat("cd", 32)
		}
		if changed, err := s.RecoverPreparationChild(ctx, o, c); err != nil || changed {
			t.Fatalf("accepted stale %s: %v %v", field, changed, err)
		}
	}
	if changed, err := s.RecoverPreparationOwner(ctx, owner); err != nil || changed {
		t.Fatalf("released pending child: %v %v", changed, err)
	}
	if changed, err := s.RecoverPreparationChild(ctx, owner, child); err != nil || !changed {
		t.Fatalf("matching recovery: %v %v", changed, err)
	}
	if changed, err := s.RecoverPreparationChild(ctx, owner, child); err != nil || changed {
		t.Fatalf("duplicate recovery: %v %v", changed, err)
	}
	stale := owner
	stale.Identity = "stale"
	if changed, err := s.RecoverPreparationOwner(ctx, stale); err != nil || changed {
		t.Fatalf("released stale owner: %v %v", changed, err)
	}
	stale = owner
	stale.Directory = filepath.Join(root, "different")
	if changed, err := s.RecoverPreparationOwner(ctx, stale); err != nil || changed {
		t.Fatalf("released changed directory: %v %v", changed, err)
	}
	// Simulate a reference appearing after candidate observation. Check both
	// ID and directory aliases without pretending this is concurrent publish.
	for _, byID := range []bool{false, true} {
		id, directory := "reference", owner.Directory
		if byID {
			id, directory = owner.ID, filepath.Join(root, "reference")
		}
		if _, err := s.db.Exec(`INSERT INTO generations(id,directory,input_digest,node_executable) VALUES(?,?,'digest','')`, id, directory); err != nil {
			t.Fatal(err)
		}
		if changed, err := s.RecoverPreparationOwner(ctx, owner); err != nil || changed {
			t.Fatalf("released referenced operation byID=%v: %v %v", byID, changed, err)
		}
		if _, err := s.db.Exec(`DELETE FROM generations WHERE id=?`, id); err != nil {
			t.Fatal(err)
		}
	}
	// Failure removing the hold must roll back the earlier status update.
	if _, err := s.db.Exec(`CREATE TRIGGER deny_hold_removal BEFORE DELETE ON operation_tree_holds BEGIN SELECT RAISE(ABORT,'injected hold failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RecoverPreparationOwner(ctx, owner); err == nil {
		t.Fatal("ignored hold-removal failure")
	}
	var status string
	if err := s.db.QueryRow(`SELECT status FROM operations WHERE id='pending'`).Scan(&status); err != nil || status != "preparing" {
		t.Fatal(status, err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER deny_hold_removal`); err != nil {
		t.Fatal(err)
	}
	if changed, err := s.RecoverPreparationOwner(ctx, owner); err != nil || !changed {
		t.Fatalf("final recovery: %v %v", changed, err)
	}
}
