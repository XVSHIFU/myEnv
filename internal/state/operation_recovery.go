package state

import "context"

type PreparationOwner struct {
	ID, Identity, Directory string
	PID                     int
	Tracked                 bool
}
type PreparationChild struct {
	ID, Token string
	Completed bool
}

// RecoverPreparationJob requires verified Windows owner death and Job absence.
func (s *Store) RecoverPreparationJob(ctx context.Context, owner PreparationOwner, child PreparationChild) (bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE operation_children SET completed=1 WHERE id=? AND operation_id=? AND completed=0
 AND NOT EXISTS(SELECT 1 FROM operation_child_completion p WHERE p.child_id=operation_children.id)
 AND EXISTS(SELECT 1 FROM operations o JOIN operation_identity i ON i.operation_id=o.id WHERE o.id=operation_children.operation_id AND o.owner_pid=? AND i.process_identity=? AND i.process_identity LIKE 'windows-v2:%' AND o.status='preparing')`, child.ID, owner.ID, owner.PID, owner.Identity)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s *Store) PreparingChildOperations(ctx context.Context, after string) ([]PreparationOwner, error) {
	var tables int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('operation_children','operation_identity','operation_child_completion')`).Scan(&tables); err != nil {
		return nil, err
	}
	if tables != 3 {
		return nil, nil
	}
	var trackedTable int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='operation_tracked'`).Scan(&trackedTable); err != nil {
		return nil, err
	}
	tracked := "0"
	if trackedTable == 1 {
		tracked = "EXISTS(SELECT 1 FROM operation_tracked t WHERE t.operation_id=o.id)"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT o.id,o.owner_pid,i.process_identity,o.directory,`+tracked+` FROM operations o JOIN operation_identity i ON i.operation_id=o.id JOIN operation_tree_holds h ON h.operation_id=o.id WHERE o.status='preparing' AND o.id>? AND (EXISTS(SELECT 1 FROM operation_children c WHERE c.operation_id=o.id) OR `+tracked+`) AND NOT EXISTS(SELECT 1 FROM generations g WHERE g.id=o.id OR g.directory=o.directory) ORDER BY o.id LIMIT 128`, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PreparationOwner
	for rows.Next() {
		var r PreparationOwner
		if err := rows.Scan(&r.ID, &r.PID, &r.Identity, &r.Directory, &r.Tracked); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *Store) PreparationChildren(ctx context.Context, operation, after string) ([]PreparationChild, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,COALESCE(p.token,''),c.completed FROM operation_children c LEFT JOIN operation_child_completion p ON p.child_id=c.id WHERE c.operation_id=? AND c.id>? ORDER BY c.id LIMIT 128`, operation, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PreparationChild
	for rows.Next() {
		var r PreparationChild
		if err := rows.Scan(&r.ID, &r.Token, &r.Completed); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// RecoverPreparationChild requires the workspace lock, an observed dead owner,
// and a valid completion receipt. All observed identity fields are compared.
func (s *Store) RecoverPreparationChild(ctx context.Context, owner PreparationOwner, child PreparationChild) (bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE operation_children SET completed=1 WHERE id=? AND operation_id=? AND completed=0
 AND EXISTS(SELECT 1 FROM operation_child_completion p WHERE p.child_id=operation_children.id AND p.token=?)
 AND EXISTS(SELECT 1 FROM operations o JOIN operation_identity i ON i.operation_id=o.id WHERE o.id=operation_children.operation_id AND o.owner_pid=? AND i.process_identity=? AND o.status='preparing')`, child.ID, owner.ID, child.Token, owner.PID, owner.Identity)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// RecoverPreparationOwner must follow owner-death verification under the lock.
// A child-free hold requires the complete-registration protocol marker.
func (s *Store) RecoverPreparationOwner(ctx context.Context, owner PreparationOwner) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE operations SET status='failed' WHERE id=? AND owner_pid=? AND directory=? AND status='preparing'
 AND EXISTS(SELECT 1 FROM operation_identity i WHERE i.operation_id=operations.id AND i.process_identity=?)
 AND EXISTS(SELECT 1 FROM operation_tree_holds h WHERE h.operation_id=operations.id)
 AND (EXISTS(SELECT 1 FROM operation_children c WHERE c.operation_id=operations.id)
 OR EXISTS(SELECT 1 FROM operation_tracked t WHERE t.operation_id=operations.id))
 AND NOT EXISTS(SELECT 1 FROM generations g WHERE g.id=operations.id OR g.directory=operations.directory)
 AND NOT EXISTS(SELECT 1 FROM operation_children c WHERE c.operation_id=operations.id AND c.completed=0)`, owner.ID, owner.PID, owner.Directory, owner.Identity)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n != 1 {
		return false, nil
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM operation_tree_holds WHERE operation_id=?`, owner.ID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
