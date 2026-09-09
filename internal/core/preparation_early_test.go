package core

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/runner"
)

func TestEarlyUVPreparationFailure(t *testing.T) {
	platform, err := runner.Platform()
	if err != nil || (platform != "windows-amd64" && platform != "linux-amd64-glibc") {
		t.Skip("requires supported descendant supervision platform")
	}
	for _, malformed := range []bool{false, true} {
		name := "missing_entry"
		if malformed {
			name = "invalid_executable"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			entry := filepath.Join(root, "invalid-uv.exe")
			if malformed {
				if err := os.WriteFile(entry, []byte("not an executable image"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			service := &Service{UV: &backend.UV{Executable: entry}}
			result, err := service.Sync(context.Background(), SyncRequest{Directory: root})
			if err == nil || errors.Is(err, runner.ErrTreeUnconfirmed) || result.Changed || result.LockChanged || result.Generation != nil {
				t.Fatalf("early failure: result=%+v err=%v", result, err)
			}
			if _, err := os.Stat(filepath.Join(root, "myenv.lock")); !os.IsNotExist(err) {
				t.Fatalf("early failure wrote declaration lock: %v", err)
			}
			db, err := sql.Open("sqlite", filepath.Join(root, ".myenv", "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			for query, want := range map[string]int{
				`SELECT count(*) FROM operations WHERE status='failed'`:     1,
				`SELECT count(*) FROM operation_tree_holds`:                 0,
				`SELECT count(*) FROM generations`:                          0,
				`SELECT count(*) FROM operation_children WHERE completed=0`: 0,
			} {
				var got int
				if err := db.QueryRow(query).Scan(&got); err != nil || got != want {
					t.Fatalf("%s: got=%d want=%d err=%v", query, got, want, err)
				}
			}
			var children int
			wantChildren := 0
			if malformed {
				wantChildren = 1
			}
			if err := db.QueryRow(`SELECT count(*) FROM operation_children`).Scan(&children); err != nil || children != wantChildren {
				t.Fatalf("early launch registration: got=%d want=%d err=%v", children, wantChildren, err)
			}
			clean, err := service.Clean(context.Background(), root, false, nil)
			if err != nil || clean.Removed != 1 {
				t.Fatalf("clean early failed preparation: %+v %v", clean, err)
			}
		})
	}
}
