package core

import (
	"fmt"
	"myenv/internal/config"
	"path/filepath"
)

// Existing workspace layout remains the migration fallback until all shared
// resource owners are connected. Explicit storage uses application directories.
func (s *Service) storagePaths(work string) (config.UserStorage, error) {
	if s.Storage == nil {
		return config.UserStorage{Data: work, Cache: filepath.Join(work, "cache")}, nil
	}
	if !filepath.IsAbs(s.Storage.Data) || !filepath.IsAbs(s.Storage.Cache) {
		return config.UserStorage{}, fmt.Errorf("shared storage directories must be absolute")
	}
	return *s.Storage, nil
}
