package core

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
)

func TestUpdatePreview(t *testing.T) {
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	suffix := map[string]string{"windows-amd64": "win-x64.zip", "linux-amd64-glibc": "linux-x64.tar.gz", "darwin-arm64": "darwin-arm64.tar.gz"}[platform]
	key := map[string]string{"windows-amd64": "win-x64-zip", "linux-amd64-glibc": "linux-x64", "darwin-arm64": "osx-arm64-tar"}[platform]
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/index.json":
			fmt.Fprintf(w, `[{"version":"v22.9.0","npm":"10.9.0","files":["%s"]}]`, key)
		case "/v22.9.0/SHASUMS256.txt":
			fmt.Fprintf(w, "%s  node-v22.9.0-%s\n", strings.Repeat("a", 64), suffix)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	if err = os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(filepath.Join(root, "myenv.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := config.Digest(c)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, "myenv.lock")
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{platform: {Tools: map[string]config.RuntimeLock{"node": {Version: "22.1.0", Backend: "node-official-v1", Evidence: "artifact", URL: server.URL + "/old", SHA256: strings.Repeat("b", 64)}}}}}
	if err = config.WriteNewLock(lockPath, lock); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	s := Service{Node: &backend.Node{Client: server.Client(), BaseURL: server.URL}}
	preview, err := s.Sync(context.Background(), SyncRequest{Directory: root, DryRun: true})
	if err != nil || preview.Plan == nil || preview.Plan.Node.Version != "22.1.0" || requests != 0 {
		t.Fatalf("retained resolution: %+v %v requests=%d", preview, err, requests)
	}
	preview, err = s.Sync(context.Background(), SyncRequest{Directory: root, DryRun: true, Update: "node"})
	if err != nil || preview.Plan == nil || preview.Plan.Node.Version != "22.9.0" || requests != 2 || preview.Changed {
		t.Fatalf("updated resolution: %+v %v requests=%d", preview, err, requests)
	}
	after, err := os.ReadFile(lockPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("preview changed lock: %v", err)
	}
	if _, err = os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatalf("preview created state: %v", err)
	}
	for _, request := range []SyncRequest{{Directory: root, Locked: true, Update: "node"}, {Directory: root, Update: "unknown"}} {
		if _, err = s.Sync(context.Background(), request); err == nil {
			t.Fatal("accepted invalid update request")
		}
	}
}
