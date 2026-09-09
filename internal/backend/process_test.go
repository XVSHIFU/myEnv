package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"myenv/internal/runner"
)

type cancelOnBackendOutput struct{ cancel context.CancelFunc }

func (w cancelOnBackendOutput) Write(p []byte) (int, error) { w.cancel(); return len(p), nil }

func TestBackendProcessCancellation(t *testing.T) {
	if os.Getenv("MYENV_BACKEND_CANCEL_HELPER") == "1" {
		fmt.Println("ready")
		time.Sleep(20 * time.Second)
		os.Exit(0)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = executeBackend(ctx, runner.Process{Executable: executable, Args: []string{"-test.run=^TestBackendProcessCancellation$"}, Environment: append(os.Environ(), "MYENV_BACKEND_CANCEL_HELPER=1"), Stdout: cancelOnBackendOutput{cancel: cancel}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation: %v", err)
	}
}
