package core

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/state"
)

func TestUseRetainsEditedDeclarationOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "injected metadata failure", 503) }))
	defer server.Close()
	root := t.TempDir()
	path := filepath.Join(root, "myenv.yaml")
	if err := os.WriteFile(path, []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s, err := state.Open(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Publish(ctx, state.Generation{ID: "old", Directory: filepath.Join(work, "old"), InputDigest: "old", NodeExecutable: filepath.Join(work, "old", "node")}, ""); err != nil {
		t.Fatal(err)
	}
	service := &Service{Node: &backend.Node{Client: server.Client(), BaseURL: server.URL}}
	result, err := service.Use(ctx, root, "node@24")
	if err == nil || !result.DeclarationChanged || !strings.Contains(err.Error(), "declaration_changed=true") {
		t.Fatalf("failure result %+v %v", result, err)
	}
	c, err := config.Load(path)
	if err != nil || c.Tools["node"] != "24" {
		t.Fatalf("lost requested declaration %+v %v", c, err)
	}
	active, err := s.Active(ctx)
	if err != nil || active.ID != "old" {
		t.Fatalf("lost active %+v %v", active, err)
	}
}

func TestUseExpectedInputRejectedBeforePreparation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	expected, _ := config.Digest(&config.Config{Schema: 1, Tools: map[string]string{"node": "24"}})
	result, err := (&Service{}).Sync(context.Background(), SyncRequest{Directory: root, ExpectedDigest: expected})
	if err == nil || !strings.HasPrefix(err.Error(), "INPUT_CHANGED:") || result.Changed || result.LockChanged {
		t.Fatalf("accepted replaced input %+v %v", result, err)
	}
	if _, err = os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatalf("stale request created preparation state: %v", err)
	}
}

func TestUseAlreadyCanceledPreservesProject(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "myenv.yaml")
	before := []byte("schema: 1\ntools: {node: '22'}\n")
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := (&Service{}).Use(ctx, root, "node@24")
	if !errors.Is(err, context.Canceled) || result.DeclarationChanged || result.Changed {
		t.Fatalf("canceled use: %+v %v", result, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("declaration changed: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "myenv.yaml" {
		t.Fatalf("canceled use created state: %v %v", entries, err)
	}
}
