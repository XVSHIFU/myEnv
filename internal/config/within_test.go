package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithinRootAndSymlinkBoundary(t *testing.T) {
	root := t.TempDir()
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{".", "./", "child/.."} {
		got, err := Within(root, relative)
		if err != nil || got != canonical {
			t.Fatalf("root %q: %q %v", relative, got, err)
		}
	}
	if _, err := Within(root, "../outside"); err == nil {
		t.Fatal("accepted lexical escape")
	}
	missing := filepath.Join("missing", "child")
	if got, err := Within(root, missing); err != nil || got != filepath.Join(canonical, missing) {
		t.Fatalf("missing child: %q %v", got, err)
	}
	t.Run("symlink", func(t *testing.T) {
		outside := t.TempDir()
		link := filepath.Join(root, "outside")
		if err := os.Symlink(outside, link); err != nil {
			t.Skipf("symlink creation unavailable: %v", err)
		}
		for _, relative := range []string{"outside", "outside/missing"} {
			if _, err := Within(root, relative); err == nil {
				t.Fatalf("accepted symlink escape %q", relative)
			}
		}
		resolved, err := filepath.EvalSymlinks(outside)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := Within(link, "."); err != nil || got != resolved {
			t.Fatalf("symlink root: %q %v", got, err)
		}
		if err := os.Remove(outside); err != nil {
			t.Fatal(err)
		}
		for _, relative := range []string{"outside", "outside/missing/child"} {
			if got, err := Within(root, relative); err == nil {
				t.Fatalf("accepted dangling link %q: %q", relative, got)
			}
		}
	})
}
