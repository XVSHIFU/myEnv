package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"myenv/internal/backend"
)

func TestPythonSyncCancellationRetained(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained uv/Python")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ UV, Python string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	declaration := filepath.Join(root, "myenv.yaml")
	if err = os.WriteFile(declaration, []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Service{UV: &backend.UV{Executable: record.UV}, PythonDirectory: filepath.Dir(filepath.Dir(record.Python))}
	first, err := s.Sync(context.Background(), SyncRequest{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		cancel()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer server.Close()
	manifest := fmt.Sprintf("[project]\nname='cancel-project'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\ndependencies=['myenv-cancel-fixture==1.0.0']\n[tool.uv]\npackage=false\n[[tool.uv.index]]\nurl='%s/simple'\ndefault=true\n", server.URL)
	if err = os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(declaration, []byte("schema: 1\ntools: {python: '3.12'}\npython: {project: '.'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Sync(ctx, SyncRequest{Directory: root})
	if !requested.Load() || !errors.Is(err, context.Canceled) || result.Changed {
		t.Fatalf("cancellation: %+v %v requested=%v", result, err, requested.Load())
	}
	selected, err := s.SelectRun(context.Background(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	defer selected.Release()
	if selected.Generation.ID != first.Generation.ID {
		t.Fatal("canceled sync published a new generation")
	}
	leftovers, err := filepath.Glob(filepath.Join(root, ".myenv-uv-config-*.toml"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("config snapshots retained: %v %v", leftovers, err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".myenv", "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var holds, failures int
	if err = db.QueryRow(`SELECT count(*) FROM operation_tree_holds`).Scan(&holds); err != nil || holds != 0 {
		t.Fatalf("confirmed success/cancellation left tree holds: %d %v", holds, err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM operations WHERE status='failed'`).Scan(&failures); err != nil || failures != 1 {
		t.Fatalf("confirmed cancellation was not recoverable: %d %v", failures, err)
	}
}
