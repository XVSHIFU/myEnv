package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerationDigestChanges(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "module.py")
	if err := os.WriteFile(file, []byte("VALUE=1"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := GenerationDigest(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := GenerationDigest(ctx, root); err != nil || again != first {
		t.Fatal("unstable baseline", err)
	}
	if err = os.Mkdir(filepath.Join(root, "__pycache__"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "__pycache__", "module.pyc"), []byte("generated"), 0600); err != nil {
		t.Fatal(err)
	}
	if again, err := GenerationDigest(ctx, root); err != nil || again != first {
		t.Fatal("bytecode changed evidence", err)
	}
	if err = os.WriteFile(file, []byte("VALUE=2"), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := GenerationDigest(ctx, root)
	if err != nil || changed == first {
		t.Fatal("missed same-size content change", err)
	}
	if err = os.WriteFile(filepath.Join(root, "extra"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	if added, err := GenerationDigest(ctx, root); err != nil || added == changed {
		t.Fatal("missed added file", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = GenerationDigest(canceled, root); err == nil {
		t.Fatal("ignored cancellation")
	}
	if _, err = GenerationDigest(canceled, filepath.Join(root, "missing")); !errors.Is(err, context.Canceled) {
		t.Fatal("opened filesystem before cancellation check", err)
	}
}
