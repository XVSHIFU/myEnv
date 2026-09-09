package state

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// BeginOperationChild records the launch before a child can execute. The caller
// holds the workspace lock; only the preparing owner may add children.
func (s *Store) BeginOperationChild(ctx context.Context, operation string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := checkOperationOwner(ctx, tx, operation); err != nil {
		return "", err
	}
	var held int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM operation_tree_holds WHERE operation_id=?`, operation).Scan(&held); err != nil {
		return "", err
	}
	if held != 1 {
		return "", fmt.Errorf("operation child-launch phase is closed")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(random[:])
	if _, err := tx.ExecContext(ctx, `INSERT INTO operation_children(id,operation_id) VALUES(?,?)`, id, operation); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

// CompleteOperationChild requires successful descendant completion observation.
func (s *Store) CompleteOperationChild(ctx context.Context, operation, child string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := checkOperationOwner(ctx, tx, operation); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE operation_children SET completed=1 WHERE id=? AND operation_id=? AND completed=0`, child, operation)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("pending operation child not found")
	}
	return tx.Commit()
}

func checkOperationChildrenDone(ctx context.Context, tx *sql.Tx, operation string) error {
	var pending int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM operation_children WHERE operation_id=? AND completed=0)`, operation).Scan(&pending); err != nil {
		return err
	}
	if pending != 0 {
		return fmt.Errorf("operation children remain unconfirmed")
	}
	return nil
}

func (s *Store) BindOperationChildCompletion(ctx context.Context, operation, child, token string) error {
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != token {
		return fmt.Errorf("invalid child completion token")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := checkOperationOwner(ctx, tx, operation); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO operation_child_completion(child_id,token) SELECT id,? FROM operation_children WHERE id=? AND operation_id=? AND completed=0`, token, child, operation)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("pending child completion owner does not match")
	}
	return tx.Commit()
}
