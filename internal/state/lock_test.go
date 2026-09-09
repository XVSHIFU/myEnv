package state

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkspaceLockContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.lock")
	release, err := LockWorkspace(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	second, err := LockWorkspace(ctx, path)
	if err == nil {
		second()
		release()
		t.Fatal("concurrent modification allowed")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		release()
		t.Fatal(err)
	}
	if err = release(); err != nil {
		t.Fatal(err)
	}
	third, err := LockWorkspace(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err = third(); err != nil {
		t.Fatal(err)
	}
}
