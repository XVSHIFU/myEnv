package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestStatusReadOnly(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	service := &Service{}
	declaration := filepath.Join(root, "myenv.yaml")
	if err := os.WriteFile(declaration, []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	check := func(want string) {
		t.Helper()
		got, err := service.Status(ctx, root)
		if err != nil || got.Environment != want {
			t.Fatalf("want %s got %+v %v", want, got, err)
		}
	}
	check("not_ready")
	work := filepath.Join(root, ".myenv")
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatalf("status created state: %v", err)
	}
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(work, "state.db")
	s, err := state.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(declaration)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := config.Digest(c)
	g := state.Generation{ID: "status", Directory: work, InputDigest: digest, NodeExecutable: filepath.Join(work, "node")}
	if err = os.WriteFile(g.NodeExecutable, []byte("not executed by status"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = s.Publish(ctx, g, ""); err != nil {
		t.Fatal(err)
	}
	s.Close()
	check("incomplete")
	if err = config.WriteSnapshot(work, c); err != nil {
		t.Fatal(err)
	}
	if err = config.MarkComplete(work, digest); err != nil {
		t.Fatal(err)
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{platform: {Tools: map[string]config.RuntimeLock{"node": {Version: "22.1.0", Backend: "node-official-v1", Evidence: "artifact", URL: "https://example.test/node", SHA256: strings.Repeat("a", 64)}}}}}
	for _, directory := range []string{root, work} {
		if err = config.WriteNewLock(filepath.Join(directory, "myenv.lock"), lock); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadFile(db)
	if err != nil {
		t.Fatal(err)
	}
	check("ready")
	after, err := os.ReadFile(db)
	if err != nil || string(before) != string(after) {
		t.Fatal("status modified database")
	}
	if err = os.WriteFile(declaration, []byte("schema: 1\ntools: {node: \"24\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	check("drifted")
}
