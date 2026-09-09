package core

import (
	"context"
	"fmt"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanMissingGenerationParent(t *testing.T) {
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
	previous := ""
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("%032x", i)
		dir := filepath.Join(work, "generations", id)
		if err = s.Publish(ctx, state.Generation{ID: id, Directory: dir, InputDigest: id, PythonExecutable: filepath.Join(dir, "python")}, previous); err != nil {
			t.Fatal(err)
		}
		previous = id
	}
	// Persisted state represents externally removed generations and operations
	// parents. Cleanup must not recreate them or remove protected metadata.
	for _, dry := range []bool{true, false} {
		result, err := (&Service{}).Clean(ctx, project, dry, nil)
		if err != nil || result.Candidates != 1 || result.Bytes != 0 || result.Changed == dry {
			t.Fatalf("missing-parent clean %+v %v", result, err)
		}
		for _, name := range []string{"generations", "operations"} {
			if _, err = os.Stat(filepath.Join(work, name)); !os.IsNotExist(err) {
				t.Fatal("cleanup recreated parent", name, err)
			}
		}
	}
	active, err := s.Active(ctx)
	if err != nil || active == nil || active.ID != fmt.Sprintf("%032x", 3) {
		t.Fatal("active metadata lost", err)
	}
	prior, err := s.Previous(ctx)
	if err != nil || prior == nil || prior.ID != fmt.Sprintf("%032x", 2) {
		t.Fatal("previous metadata lost", err)
	}
	result, err := (&Service{}).Clean(ctx, project, true, nil)
	if err != nil || result.Candidates != 0 {
		t.Fatalf("stale metadata %+v %v", result, err)
	}
}
