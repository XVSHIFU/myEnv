package state

import (
	"context"
	"fmt"
)

const preparationEligible = `o.status IN ('failed','complete') AND NOT EXISTS
 (SELECT 1 FROM generations g WHERE g.id=o.id OR g.directory=o.directory)`

// PreparationCandidates excludes all published objects, including those protected by
// active references or leases. Preparing operations never become candidates here.
type Preparation struct {
	Generation
	Status string
}

func (s *Store) PreparationCandidates(ctx context.Context, after string, limit int) ([]Preparation, error) {
	return s.preparationCandidates(ctx, after, limit, preparationEligible)
}

// PreviewPreparationCandidates also shows explicitly completed child phases.
// The owner may still publish after this read-only preview; actual clean must
// acquire the workspace lock and recover/recheck before deleting anything.
func (s *Store) PreviewPreparationCandidates(ctx context.Context, after string, limit int) ([]Preparation, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='operation_tree_completed'`).Scan(&exists); err != nil {
		return nil, err
	}
	eligible := preparationEligible
	if exists != 0 {
		eligible = `(` + preparationEligible + `) OR (o.status='preparing'
 AND EXISTS (SELECT 1 FROM operation_tree_completed c WHERE c.operation_id=o.id)
 AND NOT EXISTS (SELECT 1 FROM operation_tree_holds h WHERE h.operation_id=o.id)
 AND NOT EXISTS (SELECT 1 FROM generations g WHERE g.id=o.id OR g.directory=o.directory))`
	}
	return s.preparationCandidates(ctx, after, limit, eligible)
}

func (s *Store) preparationCandidates(ctx context.Context, after string, limit int, eligible string) ([]Preparation, error) {
	if limit < 1 || limit > 256 {
		return nil, fmt.Errorf("clean page limit must be between 1 and 256")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT o.id,o.directory,o.input_digest,o.status FROM operations o WHERE o.id>? AND (`+eligible+`) ORDER BY o.id LIMIT ?`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Preparation
	for rows.Next() {
		var g Preparation
		if err = rows.Scan(&g.ID, &g.Directory, &g.InputDigest, &g.Status); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

func (s *Store) MarkPreparationDeleting(ctx context.Context, id, directory string) (bool, error) {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO deleting_operations(operation_id) SELECT o.id FROM operations o WHERE o.id=? AND o.directory=? AND `+preparationEligible, id, directory)
	if err != nil {
		return false, err
	}
	var count int
	err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM operations o JOIN deleting_operations d ON d.operation_id=o.id WHERE o.id=? AND o.directory=? AND `+preparationEligible, id, directory).Scan(&count)
	return count == 1, err
}

// FinishPreparationDeleting follows removal of both generations/<id> and operations/<id>.
func (s *Store) FinishPreparationDeleting(ctx context.Context, id, directory string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM operations o JOIN deleting_operations d ON d.operation_id=o.id WHERE o.id=? AND o.directory=? AND `+preparationEligible, id, directory).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("preparation is not reserved for deletion")
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM deleting_operations WHERE operation_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM operations WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
