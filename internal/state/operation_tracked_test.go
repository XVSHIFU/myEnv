package state

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestTrackedOperationUpgrade(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"empty", "child"} {
		if err := s.BeginGuardedOperation(ctx, id, filepath.Join(root, id), "digest"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.BeginOperationChild(ctx, "child"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DROP TABLE operation_tracked`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ro, err := OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ro.PreparingChildOperations(ctx, "")
	if err != nil || len(owners) != 1 || owners[0].ID != "child" || owners[0].Tracked {
		t.Fatalf("old schema candidates: %+v %v", owners, err)
	}
	var n int
	if err := ro.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='operation_tracked'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("readonly migration: %d %v", n, err)
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("readonly changed bytes", err)
	}
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.db.QueryRow(`SELECT count(*) FROM operation_tracked`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("historical marker backfill: %d %v", n, err)
	}
	owners, err = s.PreparingChildOperations(ctx, "")
	if err != nil || len(owners) != 1 || owners[0].ID != "child" || owners[0].Tracked {
		t.Fatalf("upgraded candidates: %+v %v", owners, err)
	}
	var owner PreparationOwner
	if err := s.db.QueryRow(`SELECT o.id,o.owner_pid,i.process_identity,o.directory FROM operations o JOIN operation_identity i ON i.operation_id=o.id WHERE o.id='empty'`).Scan(&owner.ID, &owner.PID, &owner.Identity, &owner.Directory); err != nil {
		t.Fatal(err)
	}
	if recovered, err := s.RecoverPreparationOwner(ctx, owner); err != nil || recovered {
		t.Fatalf("recovered historical empty: %v %v", recovered, err)
	}
}

func TestTrackedOperationAtomicRegistration(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.BeginGuardedOperation(ctx, "historical", root, "digest"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_tracked BEFORE INSERT ON operation_tracked BEGIN SELECT RAISE(ABORT,'test tracked failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginTrackedOperation(ctx, "new", filepath.Join(root, "new"), "digest"); err == nil {
		t.Fatal("accepted failed protocol registration")
	}
	for _, table := range []string{"operations", "operation_identity", "operation_tree_holds"} {
		var n int
		column := "operation_id"
		if table == "operations" {
			column = "id"
		}
		if err := s.db.QueryRow(`SELECT count(*) FROM ` + table + ` WHERE ` + column + `='new'`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("partial %s registration: %d %v", table, n, err)
		}
	}
	if _, err := s.db.Exec(`DROP TRIGGER reject_tracked`); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginTrackedOperation(ctx, "new", filepath.Join(root, "new"), "digest"); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := s.db.QueryRow(`SELECT operation_id FROM operation_tracked`).Scan(&id); err != nil || id != "new" {
		t.Fatalf("protocol marker: %q %v", id, err)
	}
	owners, err := s.PreparingChildOperations(ctx, "")
	if err != nil || len(owners) != 1 || owners[0].ID != "new" || !owners[0].Tracked {
		t.Fatalf("empty tracked candidates: %+v %v", owners, err)
	}
	wrong := owners[0]
	wrong.Identity += "-stale"
	if ok, err := s.RecoverPreparationOwner(ctx, wrong); err != nil || ok {
		t.Fatalf("accepted wrong owner: %v %v", ok, err)
	}
	if ok, err := s.RecoverPreparationOwner(ctx, owners[0]); err != nil || !ok {
		t.Fatalf("tracked empty CAS: %v %v", ok, err)
	}
}
