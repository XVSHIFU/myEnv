package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOperationPublication(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(root, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	directory := filepath.Join(root, "generation")
	if err = s.BeginOperation(ctx, "one", directory, "digest"); err != nil {
		t.Fatal(err)
	}
	if err = s.Publish(ctx, Generation{ID: "one", Directory: directory, InputDigest: "changed", NodeExecutable: filepath.Join(directory, "node.exe")}, ""); err == nil {
		t.Fatal("published different operation inputs")
	}
	if err = s.Publish(ctx, Generation{ID: "one", Directory: directory, InputDigest: "digest", NodeExecutable: filepath.Join(directory, "node.exe")}, ""); err != nil {
		t.Fatal(err)
	}
	if err = s.FailOperation(ctx, "one"); err != nil {
		t.Fatal(err)
	}
	var status string
	if err = s.db.QueryRow(`SELECT status FROM operations WHERE id='one'`).Scan(&status); err != nil || status != "complete" {
		t.Fatalf("status %s %v", status, err)
	}
	if err = s.BeginOperation(ctx, "two", filepath.Join(root, "two"), "digest"); err != nil {
		t.Fatal(err)
	}
	if err = s.FailOperation(ctx, "two"); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT status FROM operations WHERE id='two'`).Scan(&status); err != nil || status != "failed" {
		t.Fatalf("status %s %v", status, err)
	}
}
