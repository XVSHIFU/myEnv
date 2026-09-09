// Package state owns the sole authoritative active-generation reference.
package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db            *sql.DB
	pythonEntries bool
}
type Generation struct {
	EmptyProfile bool `json:"-"`
	// PreparedEntry is publication-only evidence for fixed-layout SDK entries.
	// Selection reconstructs those paths from the immutable snapshot and directory.
	PreparedEntry    string `json:"-"`
	ID               string `json:"id"`
	Directory        string `json:"directory"`
	InputDigest      string `json:"input_digest"`
	NodeExecutable   string `json:"node_executable"`
	PythonExecutable string `json:"python_executable,omitempty"`
}

func Open(ctx context.Context, path string) (*Store, error) {
	return openStore(ctx, path, false)
}

// OpenReadOnly never creates a database or runs schema initialization.
func OpenReadOnly(ctx context.Context, path string) (*Store, error) {
	return openStore(ctx, path, true)
}

// OpenForRun opens an existing state file and avoids repeating schema DDL when
// all required objects are already present. Missing objects use the original
// initialization below, including the deletion-protection triggers.
func OpenForRun(ctx context.Context, path string) (*Store, error) {
	return openStoreMode(ctx, path, false, true)
}

func openStore(ctx context.Context, path string, readOnly bool) (*Store, error) {
	return openStoreMode(ctx, path, readOnly, false)
}

func openStoreMode(ctx context.Context, path string, readOnly, existingRun bool) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(absolute)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	query := u.Query()
	if readOnly {
		query.Set("mode", "ro")
	} else if existingRun {
		query.Set("mode", "rw")
	}
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "synchronous(FULL)")
	u.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if readOnly {
		// The schema query opens and validates the connection itself.
		var exists int
		if err = db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='generation_python'`).Scan(&exists); err != nil {
			db.Close()
			return nil, err
		}
		return &Store{db: db, pythonEntries: exists != 0}, nil
	}
	if existingRun {
		complete, err := hasRunSchema(ctx, db)
		if err != nil {
			db.Close()
			return nil, err
		}
		if complete {
			return &Store{db: db, pythonEntries: true}, nil
		}
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS generations (
 id TEXT PRIMARY KEY, directory TEXT NOT NULL UNIQUE, input_digest TEXT NOT NULL,
 node_executable TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS active (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 current_id TEXT REFERENCES generations(id), previous_id TEXT REFERENCES generations(id));
 INSERT OR IGNORE INTO active(singleton) VALUES(1);
 CREATE TABLE IF NOT EXISTS leases (
 id TEXT PRIMARY KEY, generation_id TEXT NOT NULL REFERENCES generations(id),
 supervisor_pid INTEGER NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
 CREATE INDEX IF NOT EXISTS leases_generation_page ON leases(generation_id,id);
 CREATE TABLE IF NOT EXISTS lease_identity (
 lease_id TEXT PRIMARY KEY REFERENCES leases(id) ON DELETE CASCADE,
 process_identity TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS lease_completion (
 lease_id TEXT PRIMARY KEY REFERENCES leases(id) ON DELETE CASCADE,
 token TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS operations (
 id TEXT PRIMARY KEY, directory TEXT NOT NULL UNIQUE, input_digest TEXT NOT NULL,
 owner_pid INTEGER NOT NULL, status TEXT NOT NULL CHECK(status IN ('preparing','complete','failed')),
 created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
 CREATE TABLE IF NOT EXISTS operation_tree_holds (
 operation_id TEXT PRIMARY KEY REFERENCES operations(id) ON DELETE CASCADE);
 CREATE TABLE IF NOT EXISTS operation_identity (
 operation_id TEXT PRIMARY KEY REFERENCES operations(id) ON DELETE CASCADE,
 process_identity TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS operation_tracked (
 operation_id TEXT PRIMARY KEY REFERENCES operations(id) ON DELETE CASCADE);
 CREATE TABLE IF NOT EXISTS operation_children (
 id TEXT PRIMARY KEY, operation_id TEXT NOT NULL REFERENCES operations(id) ON DELETE CASCADE,
 completed INTEGER NOT NULL DEFAULT 0 CHECK(completed IN (0,1)));
 CREATE INDEX IF NOT EXISTS operation_children_owner ON operation_children(operation_id,completed);
 CREATE TABLE IF NOT EXISTS operation_child_completion (
 child_id TEXT PRIMARY KEY REFERENCES operation_children(id) ON DELETE CASCADE,
 token TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS operation_tree_completed (
 operation_id TEXT PRIMARY KEY REFERENCES operations(id) ON DELETE CASCADE);
 CREATE TABLE IF NOT EXISTS generation_python (
 generation_id TEXT PRIMARY KEY REFERENCES generations(id), executable TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS generation_evidence (
 generation_id TEXT PRIMARY KEY REFERENCES generations(id) ON DELETE CASCADE,
 policy TEXT NOT NULL, digest TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS deleting_generations (
 generation_id TEXT PRIMARY KEY REFERENCES generations(id));
 CREATE TABLE IF NOT EXISTS deleting_operations (
 operation_id TEXT PRIMARY KEY REFERENCES operations(id));
 CREATE TRIGGER IF NOT EXISTS protect_deleting_preparation BEFORE INSERT ON generations
 WHEN EXISTS (SELECT 1 FROM deleting_operations d JOIN operations o ON o.id=d.operation_id
 WHERE o.id=NEW.id OR o.directory=NEW.directory)
 BEGIN SELECT RAISE(ABORT,'preparation is being deleted'); END;
 CREATE TRIGGER IF NOT EXISTS protect_deleting_operation_update BEFORE UPDATE ON operations
 WHEN EXISTS (SELECT 1 FROM deleting_operations WHERE operation_id=OLD.id)
 BEGIN SELECT RAISE(ABORT,'preparation is being deleted'); END;
 CREATE TRIGGER IF NOT EXISTS protect_deleting_active BEFORE UPDATE ON active
 WHEN EXISTS (SELECT 1 FROM deleting_generations WHERE generation_id=NEW.current_id OR generation_id=NEW.previous_id)
 BEGIN SELECT RAISE(ABORT,'generation is being deleted'); END;
 CREATE TRIGGER IF NOT EXISTS protect_deleting_lease BEFORE INSERT ON leases
 WHEN EXISTS (SELECT 1 FROM deleting_generations WHERE generation_id=NEW.generation_id)
 BEGIN SELECT RAISE(ABORT,'generation is being deleted'); END;
 CREATE TRIGGER IF NOT EXISTS protect_deleting_operation BEFORE INSERT ON operations
 WHEN NEW.status='preparing' AND EXISTS (
 SELECT 1 FROM deleting_generations d JOIN generations g ON g.id=d.generation_id
 WHERE g.id=NEW.id OR g.directory=NEW.directory)
 BEGIN SELECT RAISE(ABORT,'generation is being deleted'); END;`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, pythonEntries: true}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Active(ctx context.Context) (*Generation, error) {
	var g Generation
	query := `SELECT g.id,g.directory,g.input_digest,g.node_executable
 FROM active a JOIN generations g ON g.id=a.current_id WHERE a.singleton=1`
	destinations := []any{&g.ID, &g.Directory, &g.InputDigest, &g.NodeExecutable}
	if s.pythonEntries {
		query = `SELECT g.id,g.directory,g.input_digest,g.node_executable,COALESCE(p.executable,'')
 FROM active a JOIN generations g ON g.id=a.current_id LEFT JOIN generation_python p ON p.generation_id=g.id WHERE a.singleton=1`
		destinations = append(destinations, &g.PythonExecutable)
	}
	err := s.db.QueryRowContext(ctx, query).Scan(destinations...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// Publish commits only a fully prepared generation. expectedID is optimistic
// concurrency protection in addition to the caller's workspace modification lock.
func (s *Store) Publish(ctx context.Context, g Generation, expectedID string) error {
	return s.publish(ctx, g, expectedID, "")
}

func (s *Store) publish(ctx context.Context, g Generation, expectedID, evidence string) error {
	if g.ID == "" || !filepath.IsAbs(g.Directory) || g.InputDigest == "" || (g.NodeExecutable == "" && g.PythonExecutable == "" && g.PreparedEntry == "" && !g.EmptyProfile) || (g.NodeExecutable != "" && !filepath.IsAbs(g.NodeExecutable)) || (g.PythonExecutable != "" && !filepath.IsAbs(g.PythonExecutable)) {
		return fmt.Errorf("invalid prepared generation")
	}
	if entry := g.PreparedEntry; entry != "" {
		rel, err := filepath.Rel(g.Directory, entry)
		if err != nil || !filepath.IsAbs(entry) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid SDK generation entry")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var operationDirectory, operationDigest, operationStatus string
	opErr := tx.QueryRowContext(ctx, `SELECT directory,input_digest,status FROM operations WHERE id=?`, g.ID).Scan(&operationDirectory, &operationDigest, &operationStatus)
	if opErr != nil && !errors.Is(opErr, sql.ErrNoRows) {
		return opErr
	}
	if opErr == nil && (operationDirectory != g.Directory || operationDigest != g.InputDigest || operationStatus != "preparing") {
		return fmt.Errorf("operation does not match prepared generation")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO generations(id,directory,input_digest,node_executable) VALUES(?,?,?,?)`, g.ID, g.Directory, g.InputDigest, g.NodeExecutable)
	if err != nil {
		return err
	}
	if g.PythonExecutable != "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO generation_python(generation_id,executable) VALUES(?,?)`, g.ID, g.PythonExecutable); err != nil {
			return err
		}
	}
	if evidence != "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO generation_evidence(generation_id,policy,digest) VALUES(?,'generation-tree-v1',?)`, g.ID, evidence); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE active SET previous_id=current_id,current_id=? WHERE singleton=1 AND COALESCE(current_id,'')=?`, g.ID, expectedID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("INPUT_CHANGED: active generation changed during preparation")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE operations SET status='complete' WHERE id=? AND status='preparing'`, g.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM operation_tree_holds WHERE operation_id=?`, g.ID); err != nil {
		return err
	}
	return tx.Commit()
}
