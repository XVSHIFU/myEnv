package backend

import (
	"context"
	"errors"
	"os"
	"testing"

	"myenv/internal/runner"
)

func TestBackendCancellationKeepsTreeUnconfirmed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := executeBackend(ctx, runner.Process{
		Executable: "/bin/sh", Args: []string{"-c", "kill -KILL $PPID; printf ready"},
		Environment: os.Environ(), Stdout: cancelOnBackendOutput{cancel: cancel},
	})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, runner.ErrTreeUnconfirmed) {
		t.Fatalf("cancellation hid unconfirmed tree: %v", err)
	}
}
