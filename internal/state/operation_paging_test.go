package state

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestPreparationRecoveryPaging(t *testing.T) {
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
	// Bulk fixture setup is one transaction; this test exercises query/CAS
	// pagination, not the process-owner-death protocol.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for i := 0; i < 514; i++ {
		id := fmt.Sprintf("%032x", i)
		if _, err := tx.Exec(`INSERT INTO operations(id,directory,input_digest,owner_pid,status) VALUES(?,?,'digest',42,'preparing')`, id, filepath.Join(root, id)); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`INSERT INTO operation_identity VALUES(?,'fixture-identity')`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`INSERT INTO operation_tree_holds VALUES(?)`, id); err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			if _, err := tx.Exec(`INSERT INTO operation_tracked VALUES(?)`, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	after := ""
	seen := 0
	for _, want := range []int{128, 128, 1, 0} {
		owners, err := s.PreparingChildOperations(ctx, after)
		if err != nil || len(owners) != want {
			t.Fatalf("page: %d want=%d err=%v", len(owners), want, err)
		}
		for _, owner := range owners {
			expected := fmt.Sprintf("%032x", seen*2)
			if owner.ID != expected || !owner.Tracked {
				t.Fatalf("owner=%+v want=%s", owner, expected)
			}
			after = owner.ID
			// Removing each returned candidate from the result set must not
			// shift or skip subsequent pages, unlike offset pagination.
			if ok, err := s.RecoverPreparationOwner(ctx, owner); err != nil || !ok {
				t.Fatalf("recover: %v %v", ok, err)
			}
			seen++
		}
	}
	var retained int
	if err := s.db.QueryRow(`SELECT count(*) FROM operations WHERE status='preparing'`).Scan(&retained); err != nil || retained != 257 {
		t.Fatalf("historical records: %d %v", retained, err)
	}
	if seen != 257 {
		t.Fatalf("recovered=%d", seen)
	}
}
