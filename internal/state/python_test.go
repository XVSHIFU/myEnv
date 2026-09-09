package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPythonGenerationPublication(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	database := filepath.Join(root, "state.db")
	s, err := Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	python := Generation{ID: "python", Directory: filepath.Join(root, "one"), InputDigest: "one", PythonExecutable: filepath.Join(root, "one", "python")}
	if err = s.Publish(ctx, python, ""); err != nil {
		t.Fatal(err)
	}
	mixed := Generation{ID: "mixed", Directory: filepath.Join(root, "two"), InputDigest: "two", PythonExecutable: filepath.Join(root, "two", "python"), NodeExecutable: filepath.Join(root, "two", "node")}
	if err = s.Publish(ctx, mixed, ""); err == nil {
		t.Fatal("accepted stale publication")
	}
	if err = s.Publish(ctx, mixed, "python"); err != nil {
		t.Fatal(err)
	}
	lease, err := s.AcquireActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Generation != mixed {
		t.Fatalf("lease lost runtime entries: %+v", lease.Generation)
	}
	if err = s.ReleaseLease(ctx, lease.ID); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenReadOnly(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	active, err := s.Active(ctx)
	if err != nil || *active != mixed {
		t.Fatalf("persisted mixed generation: %+v %v", active, err)
	}
	previous, err := s.Previous(ctx)
	if err != nil || *previous != python {
		t.Fatalf("persisted Python generation: %+v %v", previous, err)
	}
}

func TestReadOnlyLegacyNodeDatabase(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	database := filepath.Join(root, "state.db")
	s, err := Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	g := Generation{ID: "node", Directory: root, InputDigest: "node", NodeExecutable: filepath.Join(root, "node")}
	if err = s.Publish(ctx, g, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DROP TABLE generation_python`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenReadOnly(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	active, err := s.Active(ctx)
	if err != nil || *active != g {
		t.Fatalf("legacy node unreadable: %+v %v", active, err)
	}
}
