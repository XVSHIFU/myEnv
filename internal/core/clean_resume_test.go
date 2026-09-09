package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/state"
)

func TestCleanResumeAfterFileRemoval(t *testing.T) {
	ctx := context.Background()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(project, ".myenv", "generations")
	if err := os.MkdirAll(parent, 0700); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(project, ".myenv", "state.db")
	s, err := state.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	previous := ""
	var first state.Generation
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("%032x", i)
		dir := filepath.Join(parent, id)
		if err = os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		g := state.Generation{ID: id, Directory: dir, InputDigest: id, PythonExecutable: filepath.Join(dir, "python")}
		if err = s.BeginOperation(ctx, id, dir, id); err != nil {
			t.Fatal(err)
		}
		opDir := filepath.Join(project, ".myenv", "operations", id)
		if err = os.MkdirAll(opDir, 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(opDir, "download"), []byte("pending"), 0600); err != nil {
			t.Fatal(err)
		}
		if err = s.Publish(ctx, g, previous); err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			first = g
		}
		previous = id
	}
	marked, err := s.MarkDeleting(ctx, first.ID, first.Directory)
	if err != nil || !marked {
		t.Fatal("reservation", err)
	}
	files, err := os.OpenRoot(parent)
	if err != nil {
		t.Fatal(err)
	}
	err = removeGeneration(ctx, files, first.ID)
	files.Close()
	if err != nil {
		t.Fatal(err)
	}
	// Simulate termination after file removal but before metadata finalization.
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	preview, err := service.Clean(ctx, project, true, nil)
	if err != nil || preview.Candidates != 1 || preview.Bytes != 7 || preview.Changed {
		t.Fatalf("resume preview %+v %v", preview, err)
	}
	cleaned, err := service.Clean(ctx, project, false, nil)
	if err != nil || cleaned.Removed != 1 || !cleaned.Changed {
		t.Fatalf("resume %+v %v", cleaned, err)
	}
	if _, err = os.Stat(filepath.Join(project, ".myenv", "operations", first.ID)); !os.IsNotExist(err) {
		t.Fatal("operation files remain", err)
	}
	for i := 2; i <= 3; i++ {
		if _, err = os.Stat(filepath.Join(project, ".myenv", "operations", fmt.Sprintf("%032x", i), "download")); err != nil {
			t.Fatal("protected operation files removed", err)
		}
	}
	s, err = state.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Reusing the removed operation ID succeeds only when its completed row
	// was removed along with the generation (the new operation stays preparing).
	if err = s.BeginOperation(ctx, first.ID, first.Directory, "new"); err != nil {
		t.Fatal("completed operation metadata remains", err)
	}
	again, err := service.Clean(ctx, project, true, nil)
	if err != nil || again.Candidates != 0 {
		t.Fatalf("reservation remains %+v %v", again, err)
	}
}
