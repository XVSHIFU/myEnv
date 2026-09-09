package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReceiptRemovalExcludesConcurrentLease(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "state.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Publish(ctx, Generation{ID: "g", Directory: root, InputDigest: "g", NodeExecutable: filepath.Join(root, "node")}, ""); err != nil {
		t.Fatal(err)
	}
	other, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, err = other.db.Exec(`PRAGMA busy_timeout=20`); err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	insert := func() error {
		_, err := other.db.Exec(`INSERT INTO leases(id,generation_id,supervisor_pid) VALUES(?,'g',?)`, id, os.Getpid())
		return err
	}
	removed, err := s.RemoveUnleasedReceipt(ctx, id, func() error {
		if err := insert(); err == nil {
			t.Fatal("lease inserted during receipt unlink")
		}
		return nil
	})
	if err != nil || !removed {
		t.Fatal("orphan not removed", removed, err)
	}
	if err = insert(); err != nil {
		t.Fatal("writer was not released", err)
	}
	called := false
	removed, err = s.RemoveUnleasedReceipt(ctx, id, func() error { called = true; return nil })
	if err != nil || removed || called {
		t.Fatal("removed receipt with registered lease", removed, called, err)
	}
	if exists, err := s.LeaseExists(ctx, id); err != nil || !exists {
		t.Fatal("existing lease changed", exists, err)
	}
}
