// Package config reads bounded, declarative project inputs without executing them.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v3"
)

const MaxInputSize = 1 << 20

var ErrNoProject = errors.New("no myEnv project found")

type Config struct {
	Schema int               `yaml:"schema" json:"schema"`
	Tools  map[string]string `yaml:"tools" json:"tools"`
	Python *Python           `yaml:"python,omitempty" json:"python,omitempty"`
	Env    map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

type Python struct {
	Project string   `yaml:"project" json:"project"`
	Groups  []string `yaml:"groups,omitempty" json:"groups,omitempty"`
}

func ReadInput(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, MaxInputSize+1))
	if err == nil && len(b) > MaxInputSize {
		err = fmt.Errorf("%s exceeds 1 MiB", path)
	}
	return b, err
}

func Load(path string) (*Config, error) {
	b, err := ReadInput(path)
	if err != nil {
		return nil, err
	}
	return Parse(b, filepath.Dir(path))
}

func Parse(b []byte, root string) (*Config, error) { return parseConfig(b, root, false) }
func parseConfig(b []byte, root string, allowEmpty bool) (*Config, error) {
	if len(b) > MaxInputSize {
		return nil, errors.New("configuration exceeds 1 MiB")
	}
	var tree yaml.Node
	d := yaml.NewDecoder(bytes.NewReader(b))
	if err := d.Decode(&tree); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := d.Decode(&extra); err != io.EOF {
		return nil, errors.New("exactly one YAML document is required")
	}
	if err := checkNode(&tree, 0); err != nil {
		return nil, err
	}
	var c Config
	strict := yaml.NewDecoder(bytes.NewReader(b))
	strict.KnownFields(true)
	if err := strict.Decode(&c); err != nil {
		return nil, err
	}
	if c.Schema != 1 {
		return nil, errors.New("schema must be 1")
	}
	if c.Tools == nil || (len(c.Tools) == 0 && !allowEmpty) {
		return nil, errors.New("tools must contain python or node")
	}
	for tool, v := range c.Tools {
		if !SupportedTool(tool) {
			return nil, fmt.Errorf("unsupported tool %q", tool)
		}
		if _, err := ParseConstraint(tool, v); err != nil {
			return nil, err
		}
	}
	envKeys := make(map[string]bool, len(c.Env))
	for k, v := range c.Env {
		if k == "" || strings.ContainsAny(k, "=\x00") || strings.ContainsRune(v, 0) {
			return nil, fmt.Errorf("invalid environment key/value %q", k)
		}
		if runtime.GOOS == "windows" {
			folded := strings.ToUpper(k)
			if envKeys[folded] {
				return nil, fmt.Errorf("duplicate case-insensitive environment key %q", folded)
			}
			envKeys[folded] = true
		}
	}
	if c.Python != nil {
		if c.Tools["python"] == "" {
			return nil, errors.New("python project requires tools.python")
		}
		if c.Python.Project == "" {
			return nil, errors.New("python.project is required")
		}
		if _, err := Within(root, c.Python.Project); err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, g := range c.Python.Groups {
			if g == "" || strings.ContainsAny(g, "\x00\r\n") || seen[g] {
				return nil, errors.New("python.groups must contain distinct nonempty names")
			}
			seen[g] = true
		}
	}
	return &c, nil
}

func checkNode(n *yaml.Node, depth int) error {
	if depth > 32 {
		return errors.New("YAML nesting exceeds 32 levels")
	}
	if n.Kind == yaml.AliasNode || n.Anchor != "" {
		return errors.New("YAML aliases and anchors are not supported")
	}
	switch n.Tag {
	case "", "!!map", "!!seq", "!!str", "!!int":
	default:
		return fmt.Errorf("unsupported YAML tag %s", n.Tag)
	}
	if n.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" || seen[k.Value] {
				return fmt.Errorf("invalid or duplicate YAML key %q", k.Value)
			}
			seen[k.Value] = true
			if depth == 1 && (k.Value == "tools" || k.Value == "env") {
				if v.Kind != yaml.MappingNode {
					return fmt.Errorf("%s must be a mapping", k.Value)
				}
				for j := 1; j < len(v.Content); j += 2 {
					if v.Content[j].Tag != "!!str" {
						return fmt.Errorf("%s values must be strings", k.Value)
					}
				}
			}
		}
	}
	for _, child := range n.Content {
		if err := checkNode(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// Within checks existing ancestors as well, so symlinks cannot escape the workspace.
func Within(root, relative string) (string, error) {
	if filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" {
		return "", errors.New("python.project must be relative")
	}
	base, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	base, err = resolveWorkspacePath(base)
	if err != nil {
		return "", err
	}
	target := filepath.Join(base, relative)
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("python.project escapes workspace")
	}
	// The resolver above already resolved the complete root. A project at that
	// root needs no second traversal of the identical path.
	if target == base {
		return target, nil
	}
	probe := target
	for {
		resolved, e := resolveWorkspacePath(probe)
		if e == nil {
			r, e := filepath.Rel(base, resolved)
			if e != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
				return "", errors.New("python.project symlink escapes workspace")
			}
			break
		}
		if !os.IsNotExist(e) {
			return "", e
		}
		// A dangling link exists even though resolving its target reports
		// not-exist. Do not walk past it and treat it as an ordinary missing
		// directory: a later target creation could redirect outside the root.
		if _, statErr := os.Lstat(probe); statErr == nil {
			return "", fmt.Errorf("cannot resolve existing workspace path: %w", e)
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", e
		}
		probe = parent
	}
	return target, nil
}

// Discover stops at VCS or explicit myEnv workspace boundaries.
func Discover(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("project context must be a directory")
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "myenv.yaml")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		for _, marker := range []string{".git", ".hg", ".myenv"} {
			if _, err := os.Lstat(filepath.Join(dir, marker)); err == nil {
				return "", ErrNoProject
			} else if !os.IsNotExist(err) {
				return "", err
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoProject
		}
		dir = parent
	}
}
