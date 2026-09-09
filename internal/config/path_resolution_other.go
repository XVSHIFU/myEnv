//go:build !windows

package config

import "path/filepath"

func resolveWorkspacePath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
