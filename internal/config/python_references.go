package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// File references are collected from dependency fields only. Platform markers
// are not evaluated here: native locking can include multiple environments.
func localPythonRequirementPaths(value any, paths *[]string) error {
	switch value := value.(type) {
	case string:
		_, reference, direct := strings.Cut(value, "@")
		if !direct {
			return nil
		}
		fields := strings.Fields(reference)
		if len(fields) == 0 {
			return nil
		}
		if !strings.HasPrefix(strings.ToLower(fields[0]), "file:") {
			return nil
		}
		u, err := url.Parse(fields[0])
		if err != nil {
			return fmt.Errorf("invalid local Python reference: %w", err)
		}
		if u.User != nil || (u.Host != "" && !strings.EqualFold(u.Host, "localhost")) || u.RawQuery != "" {
			return fmt.Errorf("local Python file reference requires an unqualified local path")
		}
		path := u.Path
		if u.Opaque != "" {
			path, err = url.PathUnescape(u.Opaque)
			if err != nil {
				return fmt.Errorf("invalid local Python reference: %w", err)
			}
		}
		if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
		if path == "" || strings.ContainsRune(path, 0) {
			return fmt.Errorf("local Python reference has an empty or invalid path")
		}
		*paths = append(*paths, filepath.FromSlash(path))
	case []any:
		for _, child := range value {
			if err := localPythonRequirementPaths(child, paths); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, child := range value {
			if err := localPythonRequirementPaths(child, paths); err != nil {
				return err
			}
		}
	}
	return nil
}
