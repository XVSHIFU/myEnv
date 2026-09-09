package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myenv/internal/state"
)

func TestCleanConfirmedPreparation(t *testing.T) {
	ctx := context.Background()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(project, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := state.Open(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("%032x", i)
		dir := filepath.Join(work, "generations", id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "partial"), []byte("123"), 0600); err != nil {
			t.Fatal(err)
		}
		if i == 3 {
			err = s.BeginOperation(ctx, id, dir, "legacy")
		} else {
			err = s.BeginGuardedOperation(ctx, id, dir, "digest")
		}
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			if err := s.ConfirmOperationTreesDone(ctx, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	service := &Service{}
	unlock, err := state.LockWorkspace(ctx, filepath.Join(work, "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	blocked, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	result, cleanErr := service.Clean(blocked, project, false, nil)
	cancel()
	unlock()
	if !errors.Is(cleanErr, context.DeadlineExceeded) || result.Changed || result.RecoveredPreparations != 0 {
		t.Fatalf("clean bypassed preparing owner's lock: %+v %v", result, cleanErr)
	}
	for _, dry := range []bool{true, false} {
		result, err := service.Clean(ctx, project, dry, nil)
		if err != nil || result.Candidates != 1 || result.Bytes != 3 || result.Changed == dry {
			t.Fatalf("dry=%v result=%+v err=%v", dry, result, err)
		}
		if dry && result.RecoveredPreparations != 0 || !dry && result.RecoveredPreparations != 1 {
			t.Fatal("incorrect recovery count", result)
		}
		if dry {
			items, err := s.PreparationCandidates(ctx, "", 128)
			if err != nil || len(items) != 0 {
				t.Fatal("preview mutated preparing state", items, err)
			}
		}
	}
	for i := 2; i <= 3; i++ {
		if _, err := os.Stat(filepath.Join(work, "generations", fmt.Sprintf("%032x", i), "partial")); err != nil {
			t.Fatal("clean removed unknown or legacy preparation", err)
		}
	}
}

func TestCleanFailedPreparation(t *testing.T) {
	ctx := context.Background()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(project, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := state.Open(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i := 1; i <= 2; i++ {
		id := fmt.Sprintf("%032x", i)
		for _, parent := range []string{"generations", "operations"} {
			dir := filepath.Join(work, parent, id)
			if err = os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, "partial"), []byte("123"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err = s.BeginGuardedOperation(ctx, id, filepath.Join(work, "generations", id), "d"); err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			if err = s.FailGuardedOperation(ctx, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	unlock, err := state.LockWorkspace(ctx, filepath.Join(work, "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	recovered, recoverErr := s.RecoverInterrupted(ctx)
	unlock()
	if recoverErr != nil || recovered != 0 {
		t.Fatalf("recovered guarded preparation: %d %v", recovered, recoverErr)
	}
	id := fmt.Sprintf("%032x", 1)
	directory := filepath.Join(work, "generations", id)
	if marked, err := s.MarkPreparationDeleting(ctx, id, directory); err != nil || !marked {
		t.Fatal("reservation", err)
	}
	// Persisted reservation must exclude publication even with no matching
	// operation ID, when another generation attempts to reuse the directory.
	if err = s.Publish(ctx, state.Generation{ID: fmt.Sprintf("%032x", 3), Directory: directory, InputDigest: "d", PythonExecutable: filepath.Join(directory, "python")}, ""); err == nil {
		t.Fatal("published reserved preparation")
	}
	service := &Service{}
	for _, dry := range []bool{true, false} {
		result, err := service.Clean(ctx, project, dry, func(item CleanItem) error {
			if item.Kind != "failed_preparation" || item.ID != id || item.Bytes != 6 {
				t.Fatalf("item %+v", item)
			}
			return nil
		})
		if err != nil || result.Candidates != 1 || result.Changed == dry {
			t.Fatalf("clean %+v %v", result, err)
		}
		if dry {
			if _, err = os.Stat(filepath.Join(directory, "partial")); err != nil {
				t.Fatal("dry-run removed preparation", err)
			}
		}
	}
	for _, parent := range []string{"generations", "operations"} {
		if _, err = os.Stat(filepath.Join(work, parent, id)); !os.IsNotExist(err) {
			t.Fatal("failed files remain")
		}
		if _, err = os.Stat(filepath.Join(work, parent, fmt.Sprintf("%032x", 2), "partial")); err != nil {
			t.Fatal("removed preparing files", err)
		}
	}
	result, err := service.Clean(ctx, project, true, nil)
	if err != nil || result.Candidates != 0 {
		t.Fatalf("stale failed record %+v %v", result, err)
	}
}
