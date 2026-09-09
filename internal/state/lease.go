package state

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
)

type Lease struct {
	ID              string
	Generation      Generation
	CompletionToken string
}

var ErrNoAppliedGeneration = errors.New("ENV_NOT_READY: no applied generation; run myenv sync")

var ErrLeaseNotOwned = errors.New("lease is missing or does not belong to this supervisor")

// AcquireActive reads the selected generation and registers protection in one
// transaction. Leases are removed only after supervised execution is finished.
func (s *Store) AcquireActive(ctx context.Context) (*Lease, error) {
	return s.acquireActive(ctx, false)
}

// AcquireActiveWithCompletion commits protection and its recovery token together.
// No child may start until this transaction and receipt preparation succeed.
func (s *Store) AcquireActiveWithCompletion(ctx context.Context) (*Lease, error) {
	return s.acquireActive(ctx, true)
}

func (s *Store) acquireActive(ctx context.Context, completion bool) (*Lease, error) {
	processIdentity, err := supervisorIdentity()
	if err != nil {
		return nil, fmt.Errorf("read supervisor identity: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var lease Lease
	query := `SELECT g.id,g.directory,g.input_digest,g.node_executable FROM active a JOIN generations g ON g.id=a.current_id WHERE a.singleton=1`
	destinations := []any{&lease.Generation.ID, &lease.Generation.Directory, &lease.Generation.InputDigest, &lease.Generation.NodeExecutable}
	if s.pythonEntries {
		query = `SELECT g.id,g.directory,g.input_digest,g.node_executable,COALESCE(p.executable,'') FROM active a JOIN generations g ON g.id=a.current_id LEFT JOIN generation_python p ON p.generation_id=g.id WHERE a.singleton=1`
		destinations = append(destinations, &lease.Generation.PythonExecutable)
	}
	err = tx.QueryRowContext(ctx, query).Scan(destinations...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoAppliedGeneration
	}
	if err != nil {
		return nil, err
	}
	identity := make([]byte, 16)
	if _, err = rand.Read(identity); err != nil {
		return nil, err
	}
	lease.ID = fmt.Sprintf("%x", identity)
	if _, err = tx.ExecContext(ctx, `INSERT INTO leases(id,generation_id,supervisor_pid) VALUES(?,?,?)`, lease.ID, lease.Generation.ID, os.Getpid()); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO lease_identity(lease_id,process_identity) VALUES(?,?)`, lease.ID, processIdentity); err != nil {
		return nil, err
	}
	if completion {
		token := make([]byte, 32)
		if _, err = rand.Read(token); err != nil {
			return nil, err
		}
		lease.CompletionToken = fmt.Sprintf("%x", token)
		if _, err = tx.ExecContext(ctx, `INSERT INTO lease_completion(lease_id,token) VALUES(?,?)`, lease.ID, lease.CompletionToken); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &lease, nil
}

func (s *Store) ReleaseLease(ctx context.Context, id string) error {
	identity, err := supervisorIdentity()
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM leases WHERE id=? AND supervisor_pid=? AND EXISTS (SELECT 1 FROM lease_identity i WHERE i.lease_id=leases.id AND i.process_identity=?)`, id, os.Getpid(), identity)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrLeaseNotOwned
	}
	return nil
}
