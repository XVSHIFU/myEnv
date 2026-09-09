//go:build !linux

package core

import (
	"context"
	"myenv/internal/state"
)

func prepareRunCompletion(context.Context, string, *state.Store, *RunEnvironment) error { return nil }

func acquireRunLease(ctx context.Context, store *state.Store) (*state.Lease, error) {
	return store.AcquireActive(ctx)
}

func openRunStore(ctx context.Context, path string) (*state.Store, error) {
	return state.Open(ctx, path)
}

func cleanLeaseReceipts(context.Context, *state.Store, string, bool, func(CleanItem) error, *CleanResult) error {
	return nil
}
