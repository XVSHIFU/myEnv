package core

import (
	"context"
	"database/sql"
	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadFailureRegistered(t *testing.T) {
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "failure", 503) }))
	defer server.Close()
	c := &config.Config{Schema: 1, Tools: map[string]string{"node": "22"}}
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}"), 0600); err != nil {
		t.Fatal(err)
	}
	digest, _ := config.Digest(c)
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{platform: {Tools: map[string]config.RuntimeLock{"node": {Version: "22.1.0", Backend: "node-official-v1", Evidence: "artifact", URL: server.URL + "/node.zip", SHA256: strings.Repeat("a", 64)}}}}}
	if err := config.WriteNewLock(filepath.Join(root, "myenv.lock"), lock); err != nil {
		t.Fatal(err)
	}
	s := Service{Node: &backend.Node{Client: server.Client(), BaseURL: server.URL}}
	if _, err := s.Sync(context.Background(), SyncRequest{Directory: root, Locked: true}); err == nil {
		t.Fatal("download unexpectedly succeeded")
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".myenv", "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM operations WHERE status='failed'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("missing failed operation: %d %v", count, err)
	}
	var id string
	if err = db.QueryRow(`SELECT id FROM operations WHERE status='failed'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".myenv", "operations", id))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("failed download left unexpected content")
	}
	// A declaration change refreshes the lock before the same download fails.
	if err = os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\nenv: {MODE: changed}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Sync(context.Background(), SyncRequest{Directory: root})
	if err == nil || result.Changed || !result.LockChanged {
		t.Fatalf("missing partial lock result: %+v %v", result, err)
	}
	updated, err := config.ReadLock(filepath.Join(root, "myenv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := config.Load(filepath.Join(root, "myenv.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	newDigest, _ := config.Digest(declaration)
	if updated.ConfigDigest != newDigest {
		t.Fatal("lock result did not reflect actual write")
	}
	result, err = s.Sync(context.Background(), SyncRequest{Directory: root})
	if err == nil || result.LockChanged {
		t.Fatalf("unchanged lock reported modified: %+v %v", result, err)
	}
}
