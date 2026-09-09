//go:build !linux

package core

import (
	"context"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func prepareChildCompletion(context.Context, string, *state.Store, string, string, *runner.Process) (func(bool) error, error) {
	return func(bool) error { return nil }, nil
}
