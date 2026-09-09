package core

import (
	"context"
	"fmt"
	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) CleanUVCache(ctx context.Context, dryRun bool, report func(CleanItem) error) (CleanResult, error) {
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
	cache := filepath.Join(storage.Cache, "uv")
	info, err := os.Lstat(cache)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if !info.IsDir() {
		return result, fmt.Errorf("CLEAN_UNSAFE_PATH: uv cache is not a directory")
	}
	root, err := os.OpenRoot(storage.Cache)
	if err != nil {
		return result, err
	}
	size, err := generationBytes(ctx, root, "uv")
	root.Close()
	if err != nil {
		return result, err
	}
	item := CleanItem{Kind: "uv_cache", ID: "uv", Directory: cache, Bytes: size}
	result.Candidates = 1
	result.Bytes = size
	if !dryRun {
		var u backend.UV
		if s.UV != nil {
			u = *s.UV
		} else {
			platform, err := runner.Platform()
			if err != nil {
				return result, err
			}
			backends := filepath.Join(storage.Data, "backends")
			data, err := config.ReadInput(filepath.Join(backends, "uv-"+backend.UVVersion+"-"+platform))
			if err != nil {
				return result, fmt.Errorf("installed uv backend required for cache cleanup: %w", err)
			}
			entry, err := config.Within(backends, strings.TrimSpace(string(data)))
			if err != nil {
				return result, err
			}
			u = backend.UV{Executable: entry}
		}
		attempted, err := u.CleanCache(ctx, cache)
		result.Changed = attempted
		if err != nil {
			return result, err
		}
		if attempted {
			result.Removed = 1
			item.Removed = true
		}
	}
	if report != nil {
		if err = report(item); err != nil {
			return result, err
		}
	}
	return result, nil
}
