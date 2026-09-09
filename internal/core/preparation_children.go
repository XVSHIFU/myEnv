package core

import (
	"context"
	"errors"

	"myenv/internal/backend"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func observePreparationChildren(ctx context.Context, store *state.Store, operation, platform, directory string) context.Context {
	return backend.WithProcessObserver(ctx, func(ctx context.Context, process *runner.Process) (func(error) error, error) {
		child, err := store.BeginOperationChild(ctx, operation)
		if err != nil {
			return nil, err
		}
		process.TreeID = child
		closeCompletion, err := prepareChildCompletion(ctx, directory, store, operation, child, process)
		if err != nil {
			// Registration succeeded but no process was launched.
			return nil, errors.Join(err, store.CompleteOperationChild(context.Background(), operation, child))
		}
		return func(runErr error) error {
			if errors.Is(runErr, runner.ErrTreeUnconfirmed) || (platform != "windows-amd64" && platform != "linux-amd64-glibc") {
				return errors.Join(runner.ErrTreeUnconfirmed, closeCompletion(false))
			}
			if err := store.CompleteOperationChild(context.Background(), operation, child); err != nil {
				return errors.Join(runner.ErrTreeUnconfirmed, err, closeCompletion(false))
			}
			return closeCompletion(true)
		}, nil
	})
}
