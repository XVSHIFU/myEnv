package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) Previous(ctx context.Context) (*Generation, error) {
	var g Generation
	err := s.db.QueryRowContext(ctx, `SELECT g.id,g.directory,g.input_digest,g.node_executable FROM active a JOIN generations g ON g.id=a.previous_id WHERE a.singleton=1`).Scan(&g.ID, &g.Directory, &g.InputDigest, &g.NodeExecutable)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if s.pythonEntries {
		if err = loadPythonEntry(ctx, s.db, &g); err != nil {
			return nil, err
		}
	}
	return &g, nil
}

// Rollback swaps only the two authoritative references. The caller holds the
// workspace modification lock and validates the retained target before calling.
func (s *Store) Rollback(ctx context.Context, currentID, previousID string) error {
	if currentID == "" || previousID == "" || currentID == previousID {
		return fmt.Errorf("invalid rollback references")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE active SET current_id=previous_id,previous_id=current_id WHERE singleton=1 AND current_id=? AND previous_id=?`, currentID, previousID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("INPUT_CHANGED: active references changed before rollback")
	}
	return nil
}
