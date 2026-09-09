package state

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

type LeaseRecord struct {
	ID, GenerationID, Identity string
	PID                        int
	CompletionToken            string
}

// HasLeases reports protection without loading optional identity/completion
// metadata or enumerating records. The leases table is part of the base schema.
func (s *Store) HasLeases(ctx context.Context) (bool, error) {
	var present bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM leases)`).Scan(&present)
	return present, err
}

func (s *Store) LeaseRecords(ctx context.Context, after string, limit int) ([]LeaseRecord, error) {
	return s.leaseRecords(ctx, after, limit, "")
}

func (s *Store) GenerationLeaseRecords(ctx context.Context, generation, after string, limit int) ([]LeaseRecord, error) {
	if generation == "" {
		return nil, fmt.Errorf("missing generation identity")
	}
	return s.leaseRecords(ctx, after, limit, generation)
}

func (s *Store) leaseRecords(ctx context.Context, after string, limit int, generation string) ([]LeaseRecord, error) {
	if limit < 1 || limit > 256 {
		return nil, fmt.Errorf("lease page limit must be between 1 and 256")
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='lease_identity'`).Scan(&exists); err != nil {
		return nil, err
	}
	identity, joins := "''", ""
	if exists != 0 {
		identity, joins = "COALESCE(i.process_identity,'')", " LEFT JOIN lease_identity i ON i.lease_id=l.id"
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='lease_completion'`).Scan(&exists); err != nil {
		return nil, err
	}
	completion := "''"
	if exists != 0 {
		completion = "COALESCE(c.token,'')"
		joins += " LEFT JOIN lease_completion c ON c.lease_id=l.id"
	}
	query := `SELECT l.id,l.generation_id,l.supervisor_pid,` + identity + `,` + completion + ` FROM leases l` + joins
	query += ` WHERE l.id>?`
	args := []any{after}
	if generation != "" {
		query += ` AND l.generation_id=?`
		args = append(args, generation)
	}
	query += ` ORDER BY l.id LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []LeaseRecord
	for rows.Next() {
		var r LeaseRecord
		if err = rows.Scan(&r.ID, &r.GenerationID, &r.PID, &r.Identity, &r.CompletionToken); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// SupervisorExited requires a validated identity format from the caller.
// A different birth identity proves that the original PID owner has exited.
func SupervisorExited(pid int, identity string) (bool, error) {
	current, err := processIdentity(pid)
	if errors.Is(err, ErrProcessGone) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return current != identity, nil
}

// DeleteObservedLease only removes the exact record whose external process/tree
// evidence was checked. It never removes a replacement or unidentified lease.
func (s *Store) DeleteObservedLease(ctx context.Context, r LeaseRecord) (bool, error) {
	if r.Identity == "" {
		return false, nil
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM leases WHERE id=? AND generation_id=? AND supervisor_pid=? AND EXISTS (SELECT 1 FROM lease_identity i WHERE i.lease_id=leases.id AND i.process_identity=?) AND COALESCE((SELECT token FROM lease_completion WHERE lease_id=leases.id),'')=?`, r.ID, r.GenerationID, r.PID, r.Identity, r.CompletionToken)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// BindLeaseCompletion is single-assignment and must commit before execution.
// It cannot bind another supervisor's lease or replace an existing proof token.
func (s *Store) BindLeaseCompletion(ctx context.Context, id, token string) error {
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != token {
		return fmt.Errorf("invalid lease completion token")
	}
	identity, err := supervisorIdentity()
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO lease_completion(lease_id,token) SELECT l.id,? FROM leases l JOIN lease_identity i ON i.lease_id=l.id WHERE l.id=? AND l.supervisor_pid=? AND i.process_identity=?`, token, id, os.Getpid(), identity)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("lease completion owner does not match")
	}
	return nil
}
