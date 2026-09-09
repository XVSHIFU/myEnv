package core

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/state"
)

func TestMalformedPreparationOwnerProtected(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	work := filepath.Join(root, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(work, "state.db")
	s, err := state.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	id := "00000000000000000000000000000001"
	if err := s.BeginTrackedOperation(ctx, id, filepath.Join(work, "generations", id), "digest"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, identity := range []string{"linux:broken", "linux:00000000-0000-0000-0000-000000000000:01", "linux:00000000-0000-0000-0000-000000000000:-1"} {
		if _, err := db.Exec(`UPDATE operation_identity SET process_identity=?`, identity); err != nil {
			t.Fatal(err)
		}
		for _, dry := range []bool{true, false} {
			result, err := (&Service{}).Clean(ctx, root, dry, nil)
			if err != nil || result.Changed || result.Candidates != 0 || result.RecoveredPreparations != 0 {
				t.Fatalf("identity=%q dry=%v: %+v %v", identity, dry, result, err)
			}
		}
	}
	var holds int
	if err := db.QueryRow(`SELECT count(*) FROM operation_tree_holds`).Scan(&holds); err != nil || holds != 1 {
		t.Fatalf("lost hold: %d %v", holds, err)
	}
}
