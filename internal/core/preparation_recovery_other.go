//go:build !linux && !windows

package core

import (
	"context"
	"myenv/internal/state"
	"os"
)

func recoverPreparationChildren(context.Context, *state.Store, string, *CleanResult) (int64, error) {
	return 0, nil
}

func previewPreparationChildren(context.Context, *state.Store, *os.Root, string, string, func(CleanItem) error, *CleanResult) error {
	return nil
}
