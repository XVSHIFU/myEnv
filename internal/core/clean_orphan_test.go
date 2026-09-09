package core

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/state"
)

func TestCleanHistoricalCompletedOperation(t *testing.T) {
	ctx := context.Background()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(project, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(work, "state.db")
	s, err := state.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	db, err := sql.Open("sqlite", database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 1; i <= 2; i++ {
		id := fmt.Sprintf("%032x", i)
		dir := filepath.Join(work, "generations", id)
		opDir := filepath.Join(work, "operations", id)
		if err = os.MkdirAll(opDir, 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(opDir, "download"), []byte("12345"), 0600); err != nil {
			t.Fatal(err)
		}
		// Historical state: complete operation survives deletion of its generation.
		if _, err = db.Exec(`INSERT INTO operations(id,directory,input_digest,owner_pid,status) VALUES(?,?,?,1,'complete')`, id, dir, id); err != nil {
			t.Fatal(err)
		}
		if i == 2 {
			// A published generation with another ID but the same directory must
			// protect the operation too, even if the historical IDs are inconsistent.
			if err = s.Publish(ctx, state.Generation{ID: fmt.Sprintf("%032x", 3), Directory: dir, InputDigest: id, PythonExecutable: filepath.Join(dir, "python")}, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, dry := range []bool{true, false} {
		result, err := (&Service{}).Clean(ctx, project, dry, func(item CleanItem) error {
			if item.Kind != "orphaned_operation" || item.Bytes != 5 {
				t.Fatalf("item %+v", item)
			}
			return nil
		})
		if err != nil || result.Candidates != 1 || result.Changed == dry {
			t.Fatalf("result %+v %v", result, err)
		}
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM operations`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("remaining %d %v", count, err)
	}
	if _, err = os.Stat(filepath.Join(work, "operations", fmt.Sprintf("%032x", 2), "download")); err != nil {
		t.Fatal("removed referenced operation", err)
	}
}
