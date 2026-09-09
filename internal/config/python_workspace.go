package config

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pelletier/go-toml/v2"
)

// PythonWorkspaceRoot locates the native lock/config root for the selected
// project. Membership is checked without scanning directories: the target is
// already known, so matching ancestor declarations is sufficient.
func PythonWorkspaceRoot(root, project string) (string, error) {
	target, err := Within(root, project)
	if err != nil {
		return "", err
	}
	candidate, err := PythonNativeWorkspaceRoot(target)
	if err != nil {
		return "", err
	}
	relativeRoot, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", err
	}
	if _, err = Within(root, relativeRoot); err != nil {
		return "", fmt.Errorf("Python workspace root is outside the myEnv workspace: %w", err)
	}
	return filepath.Clean(relativeRoot), nil
}

// PythonNativeWorkspaceRoot reads only ancestor manifests. The caller decides
// whether the returned root is within its managed workspace boundary.
func PythonNativeWorkspaceRoot(target string) (string, error) {
	if !filepath.IsAbs(target) {
		return "", fmt.Errorf("Python project must be absolute")
	}
	metadataBytes := 0
	for candidate, depth := target, 0; ; candidate, depth = filepath.Dir(candidate), depth+1 {
		if depth >= 64 {
			return "", fmt.Errorf("Python workspace ancestor search exceeds 64 directories")
		}
		data, err := ReadInput(filepath.Join(candidate, "pyproject.toml"))
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil {
			metadataBytes += len(data)
			if metadataBytes > 8<<20 {
				return "", fmt.Errorf("Python ancestor metadata exceeds 8 MiB")
			}
			var document map[string]any
			if err = toml.Unmarshal(data, &document); err != nil {
				return "", err
			}
			tool, _ := document["tool"].(map[string]any)
			uv, _ := tool["uv"].(map[string]any)
			if value, exists := uv["workspace"]; exists {
				workspace, ok := value.(map[string]any)
				if !ok {
					return "", fmt.Errorf("tool.uv.workspace must be a table")
				}
				relative, err := filepath.Rel(candidate, target)
				if err != nil {
					return "", err
				}
				if relative != "." {
					excludes, err := workspacePatterns(workspace["exclude"])
					if err != nil {
						return "", err
					}
					if matchesWorkspacePattern(excludes, filepath.ToSlash(relative)) {
						return target, nil
					}
					members, err := workspacePatterns(workspace["members"])
					if err != nil {
						return "", err
					}
					if !matchesWorkspacePattern(members, filepath.ToSlash(relative)) {
						return "", fmt.Errorf("selected Python project is not a member of parent workspace %s; add it to members or exclude it", candidate)
					}
				}
				return candidate, nil
			}
		}
		if filepath.Dir(candidate) == candidate {
			break
		}
	}
	return target, nil
}

func matchesWorkspacePattern(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if match, _ := doublestar.Match(pattern, name); match {
			return true
		}
	}
	return false
}

func pythonWorkspaceMembers(root, project string, declaration any, budget *int) ([]string, error) {
	if declaration == nil {
		return nil, nil
	}
	workspace, ok := declaration.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tool.uv.workspace must be a table")
	}
	members, err := workspacePatterns(workspace["members"])
	if err != nil {
		return nil, err
	}
	exclude, err := workspacePatterns(workspace["exclude"])
	if err != nil {
		return nil, err
	}
	surface := &workspaceGlobFS{root: root, project: project, exclude: exclude, budget: budget}
	selected := map[string]bool{}
	for _, pattern := range members {
		err := doublestar.GlobWalk(surface, pattern, func(name string, entry fs.DirEntry) error {
			if surface.excluded(name) {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if !entry.IsDir() {
				info, err := fs.Stat(surface, name)
				if err != nil {
					return err
				}
				if !info.IsDir() {
					return nil
				}
			}
			selected[filepath.Join(project, filepath.FromSlash(name))] = true
			if len(selected) > 64 {
				return fmt.Errorf("Python workspace exceeds 64 members")
			}
			return nil
		}, doublestar.WithFailOnIOErrors())
		if err != nil {
			return nil, fmt.Errorf("expand Python workspace %q: %w", pattern, err)
		}
	}
	result := make([]string, 0, len(selected))
	for member := range selected {
		result = append(result, member)
	}
	sort.Strings(result)
	return result, nil
}

func workspacePatterns(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	values, ok := value.([]any)
	if !ok || len(values) > 64 {
		return nil, fmt.Errorf("Python workspace patterns must be an array of at most 64 strings")
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		pattern, ok := value.(string)
		if !ok || pattern == "" || filepath.IsAbs(pattern) || filepath.VolumeName(pattern) != "" {
			return nil, fmt.Errorf("Python workspace patterns must be relative")
		}
		pattern = filepath.ToSlash(pattern)
		for _, part := range strings.Split(pattern, "/") {
			if part == ".." {
				return nil, fmt.Errorf("Python workspace pattern escapes its root")
			}
		}
		if !doublestar.ValidatePattern(pattern) {
			return nil, fmt.Errorf("invalid Python workspace pattern %q", pattern)
		}
		pattern = strings.TrimPrefix(pattern, "./")
		result = append(result, pattern)
	}
	return result, nil
}

// Glob receives only this constrained filesystem. Errors are propagated rather
// than silently turning an unreadable or oversized workspace into zero members.
type workspaceGlobFS struct {
	root, project string
	exclude       []string
	budget        *int
}

func (w *workspaceGlobFS) excluded(name string) bool {
	for _, pattern := range w.exclude {
		if matched, _ := doublestar.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func (w *workspaceGlobFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if *w.budget <= 0 {
		return nil, fmt.Errorf("Python workspace discovery exceeds 4096 entries/operations")
	}
	*w.budget--
	path, err := Within(w.root, filepath.Join(w.project, filepath.FromSlash(name)))
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (w *workspaceGlobFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if w.excluded(name) {
		return nil, nil
	}
	f, err := w.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	directory, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, fmt.Errorf("workspace path is not a directory: %s", name)
	}
	entries, err := directory.ReadDir(*w.budget + 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > *w.budget {
		return nil, fmt.Errorf("Python workspace discovery exceeds 4096 entries/operations")
	}
	*w.budget -= len(entries)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}
