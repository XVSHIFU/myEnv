package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPublishPreservesActiveOnConflict(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	g := Generation{ID: "one", Directory: filepath.Join(root, "one"), InputDigest: "digest-one", NodeExecutable: filepath.Join(root, "one", "node.exe")}
	if err = store.Publish(ctx, g, ""); err != nil {
		t.Fatal(err)
	}
	other := Generation{ID: "two", Directory: filepath.Join(root, "two"), InputDigest: "digest-two", NodeExecutable: filepath.Join(root, "two", "node.exe")}
	if err = store.Publish(ctx, other, ""); err == nil {
		t.Fatal("stale publication accepted")
	}
	active, err := store.Active(ctx)
	if err != nil || active == nil || active.ID != "one" {
		t.Fatalf("lost old generation: %+v %v", active, err)
	}
	if err = store.Publish(ctx, other, "one"); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	active, err = store.Active(ctx)
	if err != nil || active == nil || active.ID != "two" {
		t.Fatalf("publication did not persist: %+v %v", active, err)
	}
}
