package core

import (
	"myenv/internal/config"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnapshotHealth(t *testing.T) {
	root := t.TempDir()
	c := &config.Config{Schema: 1, Tools: map[string]string{"node": "22"}}
	digest, _ := config.Digest(c)
	generation := &state.Generation{Directory: root, InputDigest: digest}
	if snapshotHealthy(generation, root) {
		t.Fatal("missing snapshot considered healthy")
	}
	if err := config.WriteSnapshot(root, c); err != nil {
		t.Fatal(err)
	}
	if snapshotHealthy(generation, root) {
		t.Fatal("accepted missing completion marker")
	}
	if err := config.MarkComplete(root, digest); err != nil {
		t.Fatal(err)
	}
	if !snapshotHealthy(generation, root) {
		t.Fatal("valid snapshot rejected")
	}
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("schema: 1\ntools: {node: \"24\"}"), 0600); err != nil {
		t.Fatal(err)
	}
	if snapshotHealthy(generation, root) {
		t.Fatal("changed snapshot considered healthy")
	}
}

func TestRuntimeSnapshotIdentity(t *testing.T) {
	root := t.TempDir()
	digest := strings.Repeat("a", 64)
	g := &state.Generation{Directory: root, InputDigest: digest}
	runtime := config.RuntimeLock{Version: "22.1.0", Backend: "node-official-v1", Evidence: "artifact", URL: "https://example.test/node.zip", SHA256: strings.Repeat("b", 64), NPM: "10.1.0"}
	if runtimeMatches(g, "windows-amd64", runtime) {
		t.Fatal("missing snapshot matched")
	}
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{"windows-amd64": {Tools: map[string]config.RuntimeLock{"node": runtime}}}}
	if err := config.WriteNewLock(filepath.Join(root, "myenv.lock"), lock); err != nil {
		t.Fatal(err)
	}
	if !runtimeMatches(g, "windows-amd64", runtime) {
		t.Fatal("identical snapshot rejected")
	}
	for _, field := range []string{"version", "hash", "url", "npm", "backend", "evidence"} {
		changed := runtime
		switch field {
		case "version":
			changed.Version = "22.2.0"
		case "hash":
			changed.SHA256 = strings.Repeat("c", 64)
		case "url":
			changed.URL += "?changed"
		case "npm":
			changed.NPM = "10.2.0"
		case "backend":
			changed.Backend = "other"
		case "evidence":
			changed.Evidence = "version"
		}
		if runtimeMatches(g, "windows-amd64", changed) {
			t.Errorf("accepted changed %s", field)
		}
	}
	if runtimeMatches(g, "darwin-arm64", runtime) {
		t.Fatal("accepted different platform")
	}
	g.InputDigest = strings.Repeat("d", 64)
	if runtimeMatches(g, "windows-amd64", runtime) {
		t.Fatal("accepted different input digest")
	}
}
