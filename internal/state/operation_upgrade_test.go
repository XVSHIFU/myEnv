package state

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestOperationIdentityUpgrade(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	unlock, err := LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	path := filepath.Join(root, "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, id := range []string{"unknown", "settled", "failed"} {
		if err := s.BeginGuardedOperation(ctx, id, filepath.Join(root, id), "digest"); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ConfirmOperationTreesDone(ctx, "settled"); err != nil {
		t.Fatal(err)
	}
	if err := s.FailGuardedOperation(ctx, "failed"); err != nil {
		t.Fatal(err)
	}
	// Represent the immediately preceding schema, without owner identities.
	if _, err := s.db.Exec(`DROP TABLE operation_identity`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	digest := func() [32]byte {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return sha256.Sum256(data)
	}
	before := digest()
	ro, err := OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if owners, err := ro.PreparingChildOperations(ctx, ""); err != nil || len(owners) != 0 {
		t.Fatalf("legacy child preview: %+v %v", owners, err)
	}
	items, err := ro.PreviewPreparationCandidates(ctx, "", 128)
	if err != nil || len(items) != 2 {
		t.Fatalf("preview=%+v error=%v", items, err)
	}
	var count int
	if err := ro.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='operation_identity'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("readonly migrated schema: %d %v", count, err)
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}
	if digest() != before {
		t.Fatal("readonly changed database bytes")
	}
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.db.QueryRow(`SELECT count(*) FROM operation_identity`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("fabricated historical identities: %d %v", count, err)
	}
	if err := s.FailGuardedOperation(ctx, "unknown"); err == nil {
		t.Fatal("claimed historical operation")
	}
	if err := s.ConfirmOperationTreesDone(ctx, "unknown"); err == nil {
		t.Fatal("confirmed historical operation")
	}
	if recovered, err := s.RecoverCompletedPreparations(ctx); err != nil || recovered != 1 {
		t.Fatalf("recovered=%d error=%v", recovered, err)
	}
	if err := s.db.QueryRow(`SELECT count(*) FROM operation_tree_holds WHERE operation_id='unknown'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("lost historical hold: %d %v", count, err)
	}
	if err := s.BeginGuardedOperation(ctx, "new", filepath.Join(root, "new"), "digest"); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT count(*) FROM operation_identity WHERE operation_id='new'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("new identity missing: %d %v", count, err)
	}
}
