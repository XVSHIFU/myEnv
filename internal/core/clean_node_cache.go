package core

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"strings"
)

// CleanNodeCache removes transport copies only. Applied environments have their
// own files. Each deletion uses the same digest lock as download and staging.
func (s *Service) CleanNodeCache(ctx context.Context, dryRun bool, report func(CleanItem) error) (CleanResult, error) {
	var result CleanResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if s.Storage == nil {
		return result, fmt.Errorf("shared storage must be specified for cache cleanup")
	}
	storage, err := s.storagePaths("")
	if err != nil {
		return result, err
	}
	directory := filepath.Join(storage.Cache, "node-archives")
	root, err := os.OpenRoot(directory)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer root.Close()
	listing, err := root.Open(".")
	if err != nil {
		return result, err
	}
	defer listing.Close()
	for {
		batch, readErr := listing.ReadDir(128)
		if readErr != nil && readErr != io.EOF {
			return result, readErr
		}
		for _, entry := range batch {
			if err = ctx.Err(); err != nil {
				return result, err
			}
			name := entry.Name()
			if len(name) != 64 || name != strings.ToLower(name) {
				continue
			}
			if _, err = hex.DecodeString(name); err != nil {
				continue
			}
			item, err := cleanNodeCacheEntry(ctx, root, directory, name, dryRun)
			if err != nil {
				return result, err
			}
			if item == nil {
				continue
			}
			result.Candidates++
			result.Bytes += item.Bytes
			if item.Removed {
				result.Removed++
				result.Changed = true
			}
			if report != nil {
				if err = report(*item); err != nil {
					return result, err
				}
			}
		}
		if readErr == io.EOF {
			return result, nil
		}
	}
}

func cleanNodeCacheEntry(ctx context.Context, root *os.Root, directory, name string, dryRun bool) (*CleanItem, error) {
	if !dryRun {
		unlock, err := state.LockWorkspace(ctx, filepath.Join(directory, name+".lock"))
		if err != nil {
			return nil, err
		}
		defer unlock()
	}
	info, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("CLEAN_UNSAFE_PATH: shared cache entry is not a regular file")
	}
	item := &CleanItem{Kind: "node_archive", ID: name, Directory: filepath.Join(directory, name), Bytes: info.Size()}
	if !dryRun {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err = root.Remove(name); err != nil {
			return nil, err
		}
		item.Removed = true
	}
	return item, nil
}
