package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func (s *Service) Rollback(ctx context.Context, directory string) (*state.Generation, error) {
	root, err := s.resolveRoot(directory)
	if err != nil {
		return nil, err
	}
	work := s.workDirectory(root)
	database := filepath.Join(work, "state.db")
	if _, err = os.Stat(database); err != nil {
		return nil, fmt.Errorf("ENV_NOT_READY: no retained environment: %w", err)
	}
	unlock, err := state.LockWorkspace(ctx, filepath.Join(work, "modify.lock"))
	if err != nil {
		return nil, err
	}
	defer unlock()
	store, err := state.Open(ctx, database)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	current, err := store.Active(ctx)
	if err != nil {
		return nil, err
	}
	previous, err := store.Previous(ctx)
	if err != nil {
		return nil, err
	}
	if current == nil || previous == nil {
		return nil, fmt.Errorf("ENV_NOT_READY: no previous generation is retained")
	}
	if !snapshotHealthy(previous, root) {
		return nil, fmt.Errorf("ENV_NOT_READY: previous generation snapshot is incomplete")
	}
	appliedConfig, err := config.ReadSnapshot(previous.Directory, root)
	if err != nil || !entriesHealthy(previous, appliedConfig.Tools) {
		return nil, fmt.Errorf("ENV_NOT_READY: previous runtime entry is unavailable")
	}
	platform, err := runner.Platform()
	if err != nil {
		return nil, err
	}
	applied, err := config.ReadLock(filepath.Join(previous.Directory, "myenv.lock"))
	if err != nil || applied.ConfigDigest != previous.InputDigest || !platformLockHealthy(applied.Platforms[platform], appliedConfig) {
		return nil, fmt.Errorf("ENV_NOT_READY: previous runtime lock is incomplete")
	}
	if err = store.Rollback(ctx, current.ID, previous.ID); err != nil {
		return nil, err
	}
	return previous, nil
}
