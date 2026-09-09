package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
	"myenv/internal/state"
)

func TestCleanOrphanedLeaseReceipts(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	work := filepath.Join(root, ".myenv")
	receipts := filepath.Join(work, "lease-receipts")
	if err := os.MkdirAll(receipts, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := state.Open(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Publish(ctx, state.Generation{ID: "active", Directory: filepath.Join(work, "generation"), InputDigest: "d", NodeExecutable: filepath.Join(work, "node")}, ""); err != nil {
		t.Fatal(err)
	}
	lease, err := store.AcquireActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(work, leaseReceiptName(lease.ID))
	if err = os.WriteFile(protected, []byte("in use"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		if err = os.WriteFile(filepath.Join(work, leaseReceiptName(fmt.Sprintf("%032x", i))), []byte("orphan"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ignored := filepath.Join(receipts, "user-file")
	if err = os.WriteFile(ignored, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(work, leaseReceiptName(fmt.Sprintf("%032x", 4)))
	if err = os.Symlink(ignored, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(work, leaseReceiptName(fmt.Sprintf("%032x", 5)))
	if err = unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	preview, err := service.Clean(ctx, root, true, nil)
	if err != nil || preview.Candidates != 3 || preview.Bytes != 18 || preview.Removed != 0 {
		t.Fatal("preview", preview, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	defer cancel()
	partial, err := service.Clean(canceled, root, false, func(item CleanItem) error {
		if item.Kind != "orphaned_lease_receipt" || !item.Removed {
			t.Fatal("receipt report", item)
		}
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || partial.Removed != 1 || !partial.Changed {
		t.Fatal("partial cancellation", partial, err)
	}
	result, err := service.Clean(ctx, root, false, nil)
	if err != nil || result.Removed != 2 || result.Bytes != 12 {
		t.Fatal("remaining cleanup", result, err)
	}
	for _, path := range []string{protected, ignored, link, fifo} {
		if _, err = os.Lstat(path); err != nil {
			t.Fatal("removed protected/unknown file", path, err)
		}
	}
	result, err = service.Clean(ctx, root, true, nil)
	if err != nil || result.Candidates != 0 {
		t.Fatal("cleanup not exhausted", result, err)
	}
}
