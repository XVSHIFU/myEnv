package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/state"
)

func TestCleanFilesBoundary(t *testing.T) {
	parent := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "keep"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"00000000000000000000000000000001", "00000000000000000000000000000002"} {
		if err := os.Mkdir(filepath.Join(parent, id), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(parent, id, "data"), []byte("content"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, id := range []string{strings.Repeat("a", 32) + ".", strings.Repeat("a", 32) + " ", strings.Repeat("A", 32), "NUL", "OLDGEN~1", strings.Repeat("a", 31), strings.Repeat("a", 33)} {
		if _, err := generationName(parent, state.Generation{ID: id, Directory: filepath.Join(parent, id)}); err == nil {
			t.Fatalf("accepted noncanonical generation %q", id)
		}
		if err := removeGeneration(context.Background(), root, id); err == nil {
			t.Fatalf("removed noncanonical generation %q", id)
		}
	}
	for _, g := range []state.Generation{
		{ID: "..", Directory: filepath.Dir(parent)},
		{ID: "00000000000000000000000000000001", Directory: outside},
		{ID: "old/data", Directory: filepath.Join(parent, "00000000000000000000000000000001", "data")},
	} {
		if _, err := generationName(parent, g); err == nil {
			t.Fatalf("accepted unsafe path %+v", g)
		}
	}
	name, err := generationName(parent, state.Generation{ID: "00000000000000000000000000000001", Directory: filepath.Join(parent, "00000000000000000000000000000001")})
	if err != nil {
		t.Fatal(err)
	}
	// The runtime uses Root to prevent following an escape even if a managed
	// directory contains an external symlink. Some Windows accounts cannot create one.
	linkErr := os.Symlink(outside, filepath.Join(parent, "00000000000000000000000000000001", "escape"))
	if linkErr != nil {
		t.Logf("symlink fixture unavailable: %v", linkErr)
	}
	size, err := generationBytes(context.Background(), root, name)
	if err != nil || size != 7 {
		t.Fatalf("size %d %v", size, err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err = removeGeneration(canceled, root, name); err == nil {
		t.Fatal("ignored cancellation")
	}
	for _, bad := range []string{".", "..", "../old", ""} {
		if err = removeGeneration(context.Background(), root, bad); err == nil {
			t.Fatalf("removed unsafe name %q", bad)
		}
	}
	if err = removeGeneration(context.Background(), root, name); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(parent, "00000000000000000000000000000001")); !os.IsNotExist(err) {
		t.Fatal("old generation remains")
	}
	for _, path := range []string{filepath.Join(parent, "00000000000000000000000000000002", "data"), filepath.Join(outside, "keep")} {
		if _, err = os.Stat(path); err != nil {
			t.Fatalf("removed protected fixture %s: %v", path, err)
		}
	}
	if err = removeGeneration(context.Background(), root, name); err != nil {
		t.Fatal("interrupted deletion is not resumable", err)
	}
}
