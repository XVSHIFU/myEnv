package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestCleanRetainsAppliedAndLeased(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := state.Open(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	previous := ""
	var lease *state.Lease
	for _, id := range []string{"00000000000000000000000000000001", "00000000000000000000000000000002", "00000000000000000000000000000003", "00000000000000000000000000000004"} {
		directory := filepath.Join(work, "generations", id)
		if err = os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directory, "python"), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		if err = s.Publish(ctx, state.Generation{ID: id, Directory: directory, InputDigest: id, PythonExecutable: filepath.Join(directory, "python")}, previous); err != nil {
			t.Fatal(err)
		}
		if id == "00000000000000000000000000000001" {
			lease, err = s.AcquireActive(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}
		previous = id
	}
	// A failed completion check must survive connection closure as a durable
	// lease, even though the generation is no longer current or previous.
	selected := leasedRunEnvironment(s, lease)
	if err = selected.Finish(errors.Join(errors.New("completion accounting failed"), runner.ErrTreeUnconfirmed)); err != nil {
		t.Fatal(err)
	}
	s, err = state.Open(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	records, err := s.GenerationLeaseRecords(ctx, lease.Generation.ID, "", 128)
	if err != nil || len(records) != 1 || records[0].ID != lease.ID {
		t.Fatalf("unconfirmed finish lost durable lease: %+v %v", records, err)
	}
	service := &Service{}
	result, err := service.Clean(ctx, root, true, func(item CleanItem) error {
		if item.ID != "00000000000000000000000000000002" || item.Removed || item.Bytes != 7 {
			t.Fatalf("dry-run item %+v", item)
		}
		return nil
	})
	if err != nil || result.Candidates != 1 || result.Removed != 0 {
		t.Fatalf("dry-run %+v %v", result, err)
	}
	if _, err = os.Stat(filepath.Join(work, "generations", "00000000000000000000000000000002", "python")); err != nil {
		t.Fatal("dry-run removed file", err)
	}
	result, err = service.Clean(ctx, root, false, nil)
	if err != nil || result.Removed != 1 {
		t.Fatalf("clean %+v %v", result, err)
	}
	for _, id := range []string{"00000000000000000000000000000001", "00000000000000000000000000000003", "00000000000000000000000000000004"} {
		if _, err = os.Stat(filepath.Join(work, "generations", id, "python")); err != nil {
			t.Fatal("removed protected generation", id, err)
		}
	}
	if _, err = os.Stat(filepath.Join(work, "generations", "00000000000000000000000000000002")); !os.IsNotExist(err) {
		t.Fatal("candidate not removed")
	}
	if err = leasedRunEnvironment(s, lease).Finish(nil); err != nil {
		t.Fatal(err)
	}
	result, err = service.Clean(ctx, root, false, nil)
	if err != nil || result.Removed != 1 {
		t.Fatalf("released clean %+v %v", result, err)
	}
	result, err = service.Clean(ctx, root, false, nil)
	if err != nil || result.Removed != 0 {
		t.Fatalf("no-op clean %+v %v", result, err)
	}
}
