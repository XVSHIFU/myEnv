package state

import (
	"context"
	"fmt"
)

// Eligibility is rechecked in the write statement, never inferred from an older
// dry-run result. All leases remain protective until explicitly released.
const cleanUnreferenced = `NOT EXISTS (SELECT 1 FROM active a WHERE a.current_id=g.id OR a.previous_id=g.id)
 AND NOT EXISTS (SELECT 1 FROM operations o WHERE o.status='preparing' AND (o.id=g.id OR o.directory=g.directory))`
const cleanEligible = cleanUnreferenced + ` AND NOT EXISTS (SELECT 1 FROM leases l WHERE l.generation_id=g.id)`

// CleanCandidates returns a bounded page of unused generations, including
// previously marked items so interrupted filesystem deletion can be resumed.
// Callers hold the workspace modification lock during mutation and deletion.
func (s *Store) CleanCandidates(ctx context.Context, afterID string, limit int) ([]Generation, error) {
	return s.cleanCandidates(ctx, afterID, limit, cleanEligible)
}

// PreviewCandidates requires the caller to examine every candidate's leases.
func (s *Store) PreviewCandidates(ctx context.Context, afterID string, limit int) ([]Generation, error) {
	return s.cleanCandidates(ctx, afterID, limit, cleanUnreferenced)
}

func (s *Store) cleanCandidates(ctx context.Context, afterID string, limit int, eligibility string) ([]Generation, error) {
	if limit < 1 || limit > 256 {
		return nil, fmt.Errorf("clean page limit must be between 1 and 256")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT g.id,g.directory,g.input_digest,g.node_executable FROM generations g WHERE g.id>? AND `+eligibility+` ORDER BY g.id LIMIT ?`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Generation
	for rows.Next() {
		var g Generation
		if err = rows.Scan(&g.ID, &g.Directory, &g.InputDigest, &g.NodeExecutable); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// MarkDeleting reserves an eligible generation before any filesystem removal.
// Directory matching prevents acting on a stale candidate with a reused ID.
func (s *Store) MarkDeleting(ctx context.Context, id, directory string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO deleting_generations(generation_id) SELECT g.id FROM generations g WHERE g.id=? AND g.directory=? AND `+cleanEligible, id, directory)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 0 {
		return n == 1, err
	}
	var marked int
	err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM generations g JOIN deleting_generations d ON d.generation_id=g.id WHERE g.id=? AND g.directory=? AND `+cleanEligible, id, directory).Scan(&marked)
	return marked == 1, err
}

// FinishDeleting is called only after the validated directory has been removed.
// On filesystem failure the marker remains durable for the next explicit clean.
func (s *Store) FinishDeleting(ctx context.Context, id, directory string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var eligible int
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM generations g JOIN deleting_generations d ON d.generation_id=g.id WHERE g.id=? AND g.directory=? AND `+cleanEligible, id, directory).Scan(&eligible)
	if err != nil {
		return err
	}
	if eligible != 1 {
		return fmt.Errorf("generation is not reserved for deletion")
	}
	for _, query := range []string{
		`DELETE FROM generation_python WHERE generation_id=?`,
		`DELETE FROM deleting_generations WHERE generation_id=?`,
		`DELETE FROM generations WHERE id=?`,
	} {
		if _, err = tx.ExecContext(ctx, query, id); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM operations WHERE id=? AND directory=? AND status='complete'`, id, directory); err != nil {
		return err
	}
	return tx.Commit()
}
