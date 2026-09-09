package state

import (
	"context"
	"encoding/hex"
	"fmt"
)

func validLeaseID(id string) bool {
	decoded, err := hex.DecodeString(id)
	return err == nil && len(decoded) == 16 && hex.EncodeToString(decoded) == id
}

func (s *Store) LeaseExists(ctx context.Context, id string) (bool, error) {
	if !validLeaseID(id) {
		return false, fmt.Errorf("invalid lease identity")
	}
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM leases WHERE id=?`, id).Scan(&count)
	return count != 0, err
}

// RemoveUnleasedReceipt holds a SQLite write transaction across the single
// filesystem unlink. A concurrent lease insert cannot race the absence check.
// remove must not call Store or recursively delete/traverse a directory.
func (s *Store) RemoveUnleasedReceipt(ctx context.Context, id string, remove func() error) (bool, error) {
	if !validLeaseID(id) {
		return false, fmt.Errorf("invalid lease identity")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Acquire the write lock even when no matching row exists; do not rely on a
	// deferred read transaction's snapshot to exclude a concurrent INSERT.
	result, err := tx.ExecContext(ctx, `UPDATE leases SET id=id WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 0 {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	if err = remove(); err != nil {
		return false, err
	}
	// No state mutation remains to commit. Rollback releases the write lock only
	// after unlink, so even commit/I/O failures cannot misreport a DB transition.
	_ = tx.Rollback()
	return true, nil
}
