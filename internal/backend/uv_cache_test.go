package backend

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUVCleanCacheRetained(t *testing.T) {
	path := os.Getenv("MYENV_TEST_UV_PREPARED")
	if path == "" {
		t.Skip("requires retained pinned uv")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ Executable string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	cache := filepath.Join(root, "cache")
	if err = os.MkdirAll(filepath.Join(cache, "archive-v0", "fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(cache, "archive-v0", "fixture", "data"), []byte("cache fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(root, "runtime")
	if err = os.WriteFile(keep, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	u := UV{Executable: record.Executable}
	changed, err := u.CleanCache(context.Background(), cache)
	if err != nil || !changed {
		t.Fatalf("cleanup %v %v", changed, err)
	}
	if _, err = os.Stat(filepath.Join(cache, "archive-v0", "fixture", "data")); !os.IsNotExist(err) {
		t.Fatal("cache file remains", err)
	}
	if data, err = os.ReadFile(keep); err != nil || string(data) != "keep" {
		t.Fatal("changed sibling runtime", err)
	}
	if changed, err = u.CleanCache(context.Background(), filepath.Join(root, "missing")); err != nil || changed {
		t.Fatal("missing cache initialized", err)
	}
	if changed, err = u.CleanCache(context.Background(), keep); err == nil || changed {
		t.Fatal("accepted file as cache")
	}
}
