package state

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

func (s *Store) PublishWithEvidence(ctx context.Context, g Generation, expectedID, digest string) error {
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || digest != strings.ToLower(digest) {
		return fmt.Errorf("invalid generation evidence digest")
	}
	return s.publish(ctx, g, expectedID, digest)
}

// GenerationEvidence is read only and supports older stores without baselines.
// Empty means missing evidence, never successful content verification.
func (s *Store) GenerationEvidence(ctx context.Context, id string) (string, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='generation_evidence'`).Scan(&exists); err != nil {
		return "", err
	}
	if exists == 0 {
		return "", nil
	}
	var policy, digest string
	err := s.db.QueryRowContext(ctx, `SELECT policy,digest FROM generation_evidence WHERE generation_id=?`, id).Scan(&policy, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if policy != "generation-tree-v1" {
		return "", fmt.Errorf("unsupported generation evidence policy %q", policy)
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || digest != strings.ToLower(digest) {
		return "", fmt.Errorf("invalid stored generation evidence")
	}
	return digest, nil
}
