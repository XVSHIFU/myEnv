package core

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestRollbackPreservesInputs(t *testing.T) {
	for _, tool := range []string{"node", "python"} {
		t.Run(tool, func(t *testing.T) { testRollbackPreservesInputs(t, tool) })
	}
}

func testRollbackPreservesInputs(t *testing.T, tool string) {
	root := t.TempDir()
	ctx := context.Background()
	work := filepath.Join(root, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := state.Open(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	prior := ""
	for _, version := range []string{"22.1.0", "24.1.0"} {
		directory := filepath.Join(work, version)
		if err = os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		c := &config.Config{Schema: 1, Tools: map[string]string{tool: version}}
		digest, _ := config.Digest(c)
		if err = config.WriteSnapshot(directory, c); err != nil {
			t.Fatal(err)
		}
		if err = config.MarkComplete(directory, digest); err != nil {
			t.Fatal(err)
		}
		runtimeLock := config.RuntimeLock{Version: version, Backend: "node-official-v1", Evidence: "artifact", URL: "https://example.test/node", SHA256: strings.Repeat("a", 64)}
		if tool == "python" {
			runtimeLock = config.RuntimeLock{Version: version, Backend: "uv-test", Evidence: "version"}
		}
		lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{platform: {Tools: map[string]config.RuntimeLock{tool: runtimeLock}}}}
		if err = config.WriteNewLock(filepath.Join(directory, "myenv.lock"), lock); err != nil {
			t.Fatal(err)
		}
		executable := filepath.Join(directory, "node")
		if err = os.WriteFile(executable, []byte("not executed by rollback"), 0700); err != nil {
			t.Fatal(err)
		}
		generation := state.Generation{ID: version, Directory: directory, InputDigest: digest, NodeExecutable: executable}
		if tool == "python" {
			generation.NodeExecutable, generation.PythonExecutable = "", executable
		}
		if err = store.Publish(ctx, generation, prior); err != nil {
			t.Fatal(err)
		}
		prior = version
	}
	declaration := []byte("schema: 1\ntools: {" + tool + ": \"24.1.0\"}\n")
	if err = os.WriteFile(filepath.Join(root, "myenv.yaml"), declaration, 0600); err != nil {
		t.Fatal(err)
	}
	lockBytes, err := os.ReadFile(filepath.Join(work, "24.1.0", "myenv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "myenv.lock"), lockBytes, 0600); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	g, err := service.Rollback(ctx, root)
	if err != nil || g.ID != "22.1.0" {
		t.Fatalf("rollback %+v %v", g, err)
	}
	for name, want := range map[string][]byte{"myenv.yaml": declaration, "myenv.lock": lockBytes} {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("modified %s: %v", name, err)
		}
	}
	if selected, err := service.SelectRun(ctx, root, false); err == nil {
		selected.Release()
		t.Fatal("ordinary run accepted drift")
	}
	selected, err := service.SelectRun(ctx, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Generation.ID != "22.1.0" {
		t.Fatal("current did not select rollback target")
	}
	if err = selected.Release(); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(work, "24.1.0", "complete")); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Rollback(ctx, root); err == nil {
		t.Fatal("accepted incomplete target")
	}
	active, err := store.Active(ctx)
	if err != nil || active.ID != "22.1.0" {
		t.Fatalf("failed rollback changed active: %+v %v", active, err)
	}
}
