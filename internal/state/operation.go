package state

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Store) BeginOperation(ctx context.Context, id, directory, digest string) error {
	if id == "" || !filepath.IsAbs(directory) || digest == "" {
		return fmt.Errorf("invalid operation")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO operations(id,directory,input_digest,owner_pid,status) VALUES(?,?,?,?,'preparing')`, id, directory, digest, os.Getpid())
	return err
}

func (s *Store) FailOperation(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE operations SET status='failed' WHERE id=? AND owner_pid=? AND status='preparing' AND NOT EXISTS (SELECT 1 FROM operation_tree_holds WHERE operation_id=operations.id)`, id, os.Getpid())
	return err
}

// RecoverInterrupted must only be called while holding this workspace's OS
// modification lock. Neither that lock nor an absent historical tree hold
// proves descendants have exited; require the explicit completion record.
// No files are removed and the authoritative active reference is untouched.
func (s *Store) RecoverInterrupted(ctx context.Context) (int64, error) {
	return s.RecoverCompletedPreparations(ctx)
}

// BeginGuardedOperation commits the protective hold before any preparation can
// launch children. A crash leaves it preparing, including across future syncs.
func (s *Store) BeginGuardedOperation(ctx context.Context, id, directory, digest string) error {
	return s.beginGuardedOperation(ctx, id, directory, digest, false)
}

// BeginTrackedOperation requires every subsequent child launch to be registered
// with BeginOperationChild before starting. Unlike historical guarded records,
// an empty child ledger then proves that this operation launched no children.
func (s *Store) BeginTrackedOperation(ctx context.Context, id, directory, digest string) error {
	return s.beginGuardedOperation(ctx, id, directory, digest, true)
}

func (s *Store) beginGuardedOperation(ctx context.Context, id, directory, digest string, tracked bool) error {
	if id == "" || !filepath.IsAbs(directory) || digest == "" {
		return fmt.Errorf("invalid operation")
	}
	identity, err := supervisorIdentity()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO operations(id,directory,input_digest,owner_pid,status) VALUES(?,?,?,?,'preparing')`, id, directory, digest, os.Getpid()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_tree_holds(operation_id) VALUES(?)`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_identity(operation_id,process_identity) VALUES(?,?)`, id, identity); err != nil {
		return err
	}
	if tracked {
		if _, err = tx.ExecContext(ctx, `INSERT INTO operation_tracked(operation_id) VALUES(?)`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ConfirmOperationTreesDone ends the child-launching phase durably. The caller
// must hold the workspace modification lock, have confirmed every child tree's
// completion, and never launch another child for this operation. The preparing
// record remains protected by that lock until publication or crash recovery.
func (s *Store) ConfirmOperationTreesDone(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := checkOperationOwner(ctx, tx, id); err != nil {
		return err
	}
	if err := checkOperationChildrenDone(ctx, tx, id); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM operation_tree_holds WHERE operation_id IN
 (SELECT id FROM operations WHERE id=? AND owner_pid=? AND status='preparing')`, id, os.Getpid())
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("operation tree hold does not belong to the preparing caller")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_tree_completed(operation_id) VALUES(?)`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// RecoverCompletedPreparations requires the workspace modification lock and
// only accepts an explicit completed-child-phase record.
func (s *Store) RecoverCompletedPreparations(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE operations SET status='failed' WHERE status='preparing'
 AND id IN (SELECT operation_id FROM operation_tree_completed)
 AND NOT EXISTS (SELECT 1 FROM operation_tree_holds h WHERE h.operation_id=operations.id)`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// FailGuardedOperation is only valid after every started child tree has been
// confirmed finished. Unknown completion and interrupted callers retain holds.
func (s *Store) FailGuardedOperation(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := checkOperationOwner(ctx, tx, id); err != nil {
		return err
	}
	if err := checkOperationChildrenDone(ctx, tx, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM operation_tree_holds WHERE operation_id IN (SELECT id FROM operations WHERE id=? AND owner_pid=? AND status='preparing')`, id, os.Getpid()); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE operations SET status='failed' WHERE id=? AND owner_pid=? AND status='preparing'`, id, os.Getpid())
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("preparing operation does not belong to caller")
	}
	return tx.Commit()
}

func checkOperationOwner(ctx context.Context, tx *sql.Tx, id string) error {
	identity, err := supervisorIdentity()
	if err != nil {
		return err
	}
	var matched int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM operations o JOIN operation_identity i ON i.operation_id=o.id WHERE o.id=? AND o.owner_pid=? AND o.status='preparing' AND i.process_identity=?`, id, os.Getpid(), identity).Scan(&matched); err != nil {
		return err
	}
	if matched != 1 {
		return fmt.Errorf("preparing operation identity does not match caller")
	}
	return nil
}
