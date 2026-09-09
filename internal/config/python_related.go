package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// pythonRelatedDigest follows declared local sources, never imports metadata or
// scans package code. A bounded graph also makes cyclic source declarations safe.
func pythonRelatedDigest(workspace, project string, metadata map[string]pythonInputRead) (string, error) {
	files := map[string]string{}
	seen := map[string]bool{}
	metadataBytes := 0
	var archiveBytes int64
	discoveryBudget := 4096
	var visit func(string, bool) error
	visit = func(relative string, root bool) error {
		absolute, err := Within(workspace, relative)
		if err != nil {
			return err
		}
		relative = filepath.Clean(relative)
		if seen[relative] {
			return nil
		}
		if len(seen) >= 64 {
			return fmt.Errorf("Python local source graph exceeds 64 paths")
		}
		seen[relative] = true
		info, err := os.Stat(absolute)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("Python local source must be a directory or regular file")
			}
			f, err := os.Open(absolute)
			if err != nil {
				return err
			}
			hash := sha256.New()
			n, err := io.CopyBuffer(hash, io.LimitReader(f, (512<<20)-archiveBytes+1), make([]byte, 64<<10))
			f.Close()
			if err != nil {
				return err
			}
			archiveBytes += n
			if archiveBytes > 512<<20 {
				return fmt.Errorf("Python local source archives exceed 512 MiB")
			}
			files[filepath.ToSlash(relative)] = hex.EncodeToString(hash.Sum(nil))
			return nil
		}
		var document map[string]any
		for _, name := range []string{"pyproject.toml", "setup.cfg", "setup.py", "uv.toml"} {
			relativePath := filepath.Clean(filepath.Join(relative, name))
			read, cached := metadata[relativePath]
			if !cached {
				path, err := Within(workspace, relativePath)
				if err != nil {
					return err
				}
				read.data, read.err = ReadInput(path)
			}
			data, err := read.data, read.err
			key := filepath.ToSlash(filepath.Join(relative, name))
			if os.IsNotExist(err) {
				if !root {
					files[key] = "missing"
				}
				continue
			}
			if err != nil {
				return err
			}
			metadataBytes += len(data)
			if metadataBytes > 8<<20 {
				return fmt.Errorf("Python local source metadata exceeds 8 MiB")
			}
			// The top-level pyproject/uv.toml already have dedicated digests.
			if !root || name == "setup.cfg" || name == "setup.py" {
				hash := sha256.Sum256(data)
				files[key] = hex.EncodeToString(hash[:])
			}
			if name == "pyproject.toml" {
				if err = toml.Unmarshal(data, &document); err != nil {
					return fmt.Errorf("read Python source metadata %s: %w", key, err)
				}
			}
		}
		tool, _ := document["tool"].(map[string]any)
		uv, _ := tool["uv"].(map[string]any)
		members, err := pythonWorkspaceMembers(workspace, relative, uv["workspace"], &discoveryBudget)
		if err != nil {
			return err
		}
		for _, member := range members {
			if err = visit(member, false); err != nil {
				return err
			}
		}
		var paths []string
		localPythonSourcePaths(uv["sources"], &paths)
		metadata, _ := document["project"].(map[string]any)
		for _, requirements := range []any{metadata["dependencies"], metadata["optional-dependencies"], document["dependency-groups"], uv["dev-dependencies"], uv["override-dependencies"], uv["constraint-dependencies"]} {
			if err = localPythonRequirementPaths(requirements, &paths); err != nil {
				return err
			}
		}
		for _, source := range paths {
			target := filepath.Join(relative, source)
			if filepath.IsAbs(source) {
				target, err = filepath.Rel(workspace, source)
				if err != nil {
					return err
				}
			}
			if err = visit(target, false); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(project, true); err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", nil
	}
	data, err := json.Marshal(files)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func localPythonSourcePaths(value any, paths *[]string) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "path" {
				if path, ok := child.(string); ok && path != "" {
					*paths = append(*paths, path)
				}
			} else {
				localPythonSourcePaths(child, paths)
			}
		}
	case []any:
		for _, child := range value {
			localPythonSourcePaths(child, paths)
		}
	}
}
