package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"myenv/internal/backend"
	"myenv/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNodeArchiveSharedCache(t *testing.T) {
	data := []byte("archive transport fixture")
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.Write(data) }))
	defer server.Close()
	root := t.TempDir()
	storage := config.UserStorage{Data: filepath.Join(root, "data"), Cache: filepath.Join(root, "cache")}
	s := &Service{Storage: &storage}
	node := &backend.Node{Client: server.Client()}
	artifact := backend.NodeArtifact{URL: server.URL, SHA256: digest}
	type outcome struct {
		path string
		err  error
	}
	results := make(chan outcome, 2)
	for i := 0; i < 2; i++ {
		dir := t.TempDir()
		go func() {
			path, err := s.nodeArchive(context.Background(), node, artifact, dir)
			results <- outcome{path, err}
		}()
	}
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.path == b.path || requests.Load() != 1 {
		t.Fatalf("copies %+v %+v requests=%d", a, b, requests.Load())
	}
	if err := os.WriteFile(a.path, []byte("project change"), 0600); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(storage.Cache, "node-archives", digest)
	actual, err := os.ReadFile(cache)
	if err != nil || string(actual) != string(data) {
		t.Fatal("project modified shared cache", err)
	}
	if err = os.WriteFile(cache, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err = s.nodeArchive(context.Background(), node, artifact, dir); err == nil || !strings.Contains(err.Error(), "CHECKSUM_MISMATCH") {
		t.Fatal("accepted corrupt cache", err)
	}
	if requests.Load() != 1 {
		t.Fatal("silently downloaded after corruption")
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 0 {
		t.Fatal("left unverified staged file", err)
	}
	preview, err := s.CleanNodeCache(context.Background(), true, nil)
	if err != nil || preview.Candidates != 1 || preview.Changed {
		t.Fatalf("cache preview %+v %v", preview, err)
	}
	if _, err = os.Stat(cache); err != nil {
		t.Fatal("preview removed cache", err)
	}
	cleaned, err := s.CleanNodeCache(context.Background(), false, nil)
	if err != nil || cleaned.Removed != 1 {
		t.Fatalf("cache cleanup %+v %v", cleaned, err)
	}
	if _, err = s.nodeArchive(context.Background(), node, artifact, dir); err != nil || requests.Load() != 2 {
		t.Fatal("explicit cache repair failed", err)
	}
}

func TestNodeCacheRetainedArchive(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if path == "" {
		t.Skip("requires retained official Node archive")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Artifact backend.NodeArtifact
		Archive  string
	}
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, record.Archive) }))
	defer server.Close()
	root := t.TempDir()
	storage := config.UserStorage{Data: filepath.Join(root, "data"), Cache: filepath.Join(root, "cache")}
	s := &Service{Storage: &storage}
	node := &backend.Node{Client: server.Client()}
	artifact := record.Artifact
	artifact.URL = server.URL
	staged, err := s.nodeArchive(context.Background(), node, artifact, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	reused, err := s.nodeArchive(context.Background(), node, artifact, t.TempDir())
	if err != nil {
		t.Fatal("cache hit required network", err)
	}
	if staged == reused {
		t.Fatal("staging path reused")
	}
	prepare := backend.PrepareNodeZIP
	if record.Artifact.Platform == "linux-amd64-glibc" {
		prepare = backend.PrepareNodeTarGZ
	}
	executable, err := prepare(context.Background(), reused, filepath.Join(root, "generation"), record.Artifact)
	if err != nil || executable == "" {
		t.Fatal("cached official archive failed runtime preparation", err)
	}
}
