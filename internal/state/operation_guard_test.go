package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCompletedPreparationPublishedAfterPreview(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	directory := filepath.Join(root, "generation")
	if err = s.BeginGuardedOperation(ctx, "pending", directory, "digest"); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfirmOperationTreesDone(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewPreparationCandidates(ctx, "", 1)
	if err != nil || len(preview) != 1 || preview[0].ID != "pending" {
		t.Fatalf("missing preview: %+v %v", preview, err)
	}
	if page, err := s.PreviewPreparationCandidates(ctx, "pending", 1); err != nil || len(page) != 0 {
		t.Fatalf("preview cursor repeated entry: %+v %v", page, err)
	}
	generation := Generation{ID: "pending", Directory: directory, InputDigest: "digest", NodeExecutable: filepath.Join(directory, "node")}
	if err = s.Publish(ctx, generation, ""); err != nil {
		t.Fatal(err)
	}
	if count, err := s.RecoverCompletedPreparations(ctx); err != nil || count != 0 {
		t.Fatalf("recovered published preparation: %d %v", count, err)
	}
	if marked, err := s.MarkPreparationDeleting(ctx, preview[0].ID, preview[0].Directory); err != nil || marked {
		t.Fatalf("reserved published preparation from stale preview: %v %v", marked, err)
	}
	if items, err := s.PreviewPreparationCandidates(ctx, "", 1); err != nil || len(items) != 0 {
		t.Fatalf("published preparation still in preview: %+v %v", items, err)
	}
	if active, err := s.Active(ctx); err != nil || active == nil || *active != generation {
		t.Fatalf("changed active generation: %+v %v", active, err)
	}
}

func TestPreparationPreviewWithoutCompletionTable(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, id := range []string{"failed", "legacy_preparing"} {
		if err = s.BeginOperation(ctx, id, filepath.Join(root, id), "digest"); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.FailOperation(ctx, "failed"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DROP TABLE operation_tree_completed`); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	items, err := ro.PreviewPreparationCandidates(ctx, "", 128)
	if err != nil || len(items) != 1 || items[0].ID != "failed" {
		t.Fatalf("legacy preview: %+v %v", items, err)
	}
	var tables int
	if err = ro.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='operation_tree_completed'`).Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("preview migrated legacy schema: %d %v", tables, err)
	}
}

func TestConfirmOperationTreesDoneOwnership(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.BeginGuardedOperation(ctx, "pending", filepath.Join(t.TempDir(), "generation"), "digest"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`UPDATE operations SET owner_pid=-1 WHERE id='pending'`); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfirmOperationTreesDone(ctx, "pending"); err == nil {
		t.Fatal("confirmed another owner's operation")
	}
	var count int
	if err = s.db.QueryRow(`SELECT count(*) FROM operation_tree_holds`).Scan(&count); err != nil || count != 1 {
		t.Fatal("lost another owner's hold", count, err)
	}
	if _, err = s.db.Exec(`UPDATE operations SET owner_pid=? WHERE id='pending'`, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfirmOperationTreesDone(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	if candidates, err := s.PreparationCandidates(ctx, "", 128); err != nil || len(candidates) != 0 {
		t.Fatal("confirmation made live preparation deletable", candidates, err)
	}
	if err = s.ConfirmOperationTreesDone(ctx, "pending"); err == nil {
		t.Fatal("accepted duplicate confirmation")
	}
}

func TestFailGuardedOperationOwnership(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.BeginGuardedOperation(ctx, "pending", filepath.Join(root, "pending"), "digest"); err != nil {
		t.Fatal(err)
	}
	identity, err := supervisorIdentity()
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{
		`UPDATE operation_identity SET process_identity='stale-process' WHERE operation_id='pending'`,
		`DELETE FROM operation_identity WHERE operation_id='pending'`,
	} {
		if _, err := s.db.Exec(mutation); err != nil {
			t.Fatal(err)
		}
		if err := s.ConfirmOperationTreesDone(ctx, "pending"); err == nil {
			t.Fatal("confirmed missing or stale process identity")
		}
		if err := s.FailGuardedOperation(ctx, "pending"); err == nil {
			t.Fatal("released missing or stale process identity")
		}
		var holds int
		if err := s.db.QueryRow(`SELECT count(*) FROM operation_tree_holds WHERE operation_id='pending'`).Scan(&holds); err != nil || holds != 1 {
			t.Fatalf("lost identity-protected hold: %d %v", holds, err)
		}
	}
	if _, err := s.db.Exec(`INSERT INTO operation_identity(operation_id,process_identity) VALUES('pending',?)`, identity); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE operations SET owner_pid=-1 WHERE id='pending'`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"missing", "pending"} {
		if err := s.FailGuardedOperation(ctx, id); err == nil {
			t.Fatalf("accepted unauthorized operation %q", id)
		}
	}
	var holds int
	var status string
	if err := s.db.QueryRow(`SELECT count(*) FROM operation_tree_holds WHERE operation_id='pending'`).Scan(&holds); err != nil || holds != 1 {
		t.Fatalf("holds=%d error=%v", holds, err)
	}
	if err := s.db.QueryRow(`SELECT status FROM operations WHERE id='pending'`).Scan(&status); err != nil || status != "preparing" {
		t.Fatalf("status=%s error=%v", status, err)
	}
	if _, err := s.db.Exec(`UPDATE operations SET owner_pid=? WHERE id='pending'`, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmOperationTreesDone(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	if err := s.FailGuardedOperation(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	if err := s.FailGuardedOperation(ctx, "pending"); err == nil {
		t.Fatal("accepted already failed operation")
	}
}

func TestGuardedOperationRecovery(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "pending")
	if err = s.BeginGuardedOperation(ctx, "pending", directory, "digest"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.FailOperation(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	if count, err := s.RecoverInterrupted(ctx); err != nil || count != 0 {
		t.Fatalf("recovered uncertain operation: %d %v", count, err)
	}
	if candidates, err := s.PreparationCandidates(ctx, "", 128); err != nil || len(candidates) != 0 {
		t.Fatalf("uncertain candidates: %+v %v", candidates, err)
	}
	if marked, err := s.MarkPreparationDeleting(ctx, "pending", directory); err != nil || marked {
		t.Fatalf("reserved uncertain operation: %v %v", marked, err)
	}
	if err = s.FailGuardedOperation(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	if candidates, err := s.PreparationCandidates(ctx, "", 128); err != nil || len(candidates) != 1 {
		t.Fatalf("confirmed failure unavailable: %+v %v", candidates, err)
	}
}

func TestGuardedPublicationAtomicity(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	dir := filepath.Join(root, "pending")
	if err = s.BeginGuardedOperation(ctx, "pending", dir, "digest"); err != nil {
		t.Fatal(err)
	}
	g := Generation{ID: "pending", Directory: dir, InputDigest: "digest", NodeExecutable: filepath.Join(dir, "node")}
	if err = s.Publish(ctx, g, "wrong-active"); err == nil {
		t.Fatal("published conflicting active")
	}
	var count int
	if err = s.db.QueryRow(`SELECT count(*) FROM operation_tree_holds`).Scan(&count); err != nil || count != 1 {
		t.Fatal("failed publication lost hold", count, err)
	}
	if err = s.Publish(ctx, g, ""); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT count(*) FROM operation_tree_holds`).Scan(&count); err != nil || count != 0 {
		t.Fatal("successful publication retained hold", count, err)
	}
}
