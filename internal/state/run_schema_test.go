package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSchemaInventory(t *testing.T) {
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var missing int
	err = s.db.QueryRow(runSchemaObjects + `SELECT count(*) FROM sqlite_master s
 WHERE s.name NOT LIKE 'sqlite_%' AND NOT EXISTS (
 SELECT 1 FROM required r WHERE r.name=s.name AND r.type=s.type)`).Scan(&missing)
	if err != nil || missing != 0 {
		t.Fatal("schema inventory needs update", missing, err)
	}
	complete, err := hasRunSchema(context.Background(), s.db)
	if err != nil || !complete {
		t.Fatal("fresh schema incomplete", complete, err)
	}
}

func TestOpenForRunSkipsInitialization(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.db.Exec(`CREATE TRIGGER reject_reinitialization BEFORE INSERT ON active BEGIN SELECT RAISE(ABORT,'unexpected initialization'); END`)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenForRun(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var synchronous, foreignKeys int
	if err := s.db.QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil || synchronous != 2 {
		t.Fatal("FULL changed", synchronous, err)
	}
	if err := s.db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatal("foreign keys changed", foreignKeys, err)
	}
}

func TestOpenForRunRestoresDeletionProtection(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Publish(ctx, Generation{ID: "one", Directory: root, InputDigest: "one", NodeExecutable: filepath.Join(root, "node")}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER protect_deleting_lease; INSERT INTO deleting_generations VALUES('one')`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenForRun(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.AcquireActiveWithCompletion(ctx); err == nil {
		t.Fatal("leased a generation being deleted")
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM leases`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial lease", count, err)
	}
}

func TestOpenForRunMissingStateDoesNotCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if s, err := OpenForRun(context.Background(), path); err == nil {
		s.Close()
		t.Fatal("created missing state")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("created state file", err)
	}
}
