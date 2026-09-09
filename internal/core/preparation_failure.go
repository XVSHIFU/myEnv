package core

import (
	"context"
	"errors"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func finishFailedPreparation(store *state.Store, id, platform string, failure error) error {
	if failure == nil || errors.Is(failure, runner.ErrTreeUnconfirmed) {
		return failure
	}
	// Absence of an unconfirmed error is evidence only on platforms whose
	// runner actually waits for descendants. Others retain the preparing hold.
	if platform != "windows-amd64" && platform != "linux-amd64-glibc" {
		return errors.Join(failure, runner.ErrTreeUnconfirmed)
	}
	return errors.Join(failure, store.FailGuardedOperation(context.Background(), id))
}
