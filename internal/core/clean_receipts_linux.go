package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"myenv/internal/state"
)

func cleanLeaseReceipts(ctx context.Context, store *state.Store, work string, dry bool, report func(CleanItem) error, result *CleanResult) error {
	root, err := os.OpenRoot(work)
	if err != nil {
		return err
	}
	defer root.Close()
	receipts, err := root.OpenRoot("lease-receipts")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer receipts.Close()
	directory, err := receipts.Open(".")
	if err != nil {
		return err
	}
	defer directory.Close()
	visited := 0
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		entries, readErr := directory.ReadDir(128)
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		for _, entry := range entries {
			visited++
			if visited > 1_000_000 {
				return fmt.Errorf("lease receipt directory exceeds entry limit")
			}
			if err = ctx.Err(); err != nil {
				return err
			}
			name := entry.Name()
			id := strings.TrimSuffix(name, ".complete")
			if name != id+".complete" || !managedGenerationID(id) {
				continue
			}
			info, err := receipts.Lstat(name)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			item := CleanItem{Kind: "orphaned_lease_receipt", ID: id, Directory: filepath.Join(work, "lease-receipts", name), Bytes: info.Size()}
			if dry {
				exists, err := store.LeaseExists(ctx, id)
				if err != nil {
					return err
				}
				if exists {
					continue
				}
			} else {
				removed, err := store.RemoveUnleasedReceipt(ctx, id, func() error { return receipts.Remove(name) })
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return err
				}
				if !removed {
					continue
				}
				item.Removed = true
				result.Removed++
				result.Changed = true
			}
			result.Candidates++
			result.Bytes += item.Bytes
			if report != nil {
				if err = report(item); err != nil {
					return err
				}
			}
		}
		if readErr == io.EOF {
			return nil
		}
	}
}
