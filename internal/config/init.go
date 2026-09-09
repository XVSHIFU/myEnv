package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

type NeedsInput struct{ Message string }

func (e *NeedsInput) Error() string { return e.Message }

// Init detects static declarations and exclusively creates a configuration.
// Existing configurations are returned unchanged, including their comments.
func Init(dir string) (*Config, bool, error) {
	return InitWithInput(dir, nil)
}

// SelectVersion supplies only missing or conflicting information; nil never prompts.
// tool is empty when the user must choose both the tool and its version.
type SelectVersion func(tool, reason string) (selectedTool, version string, err error)

func InitWithInput(dir string, selectVersion SelectVersion) (*Config, bool, error) {
	path := filepath.Join(dir, "myenv.yaml")
	if _, err := os.Lstat(path); err == nil {
		c, e := Load(path)
		return c, false, e
	} else if !os.IsNotExist(err) {
		return nil, false, err
	}
	c := &Config{Schema: 1, Tools: map[string]string{}}
	choose := func(tool, reason string) (string, string, error) {
		if selectVersion == nil {
			return "", "", &NeedsInput{reason}
		}
		selected, version, err := selectVersion(tool, reason)
		if err != nil {
			return "", "", err
		}
		if tool != "" && selected != tool {
			return "", "", fmt.Errorf("expected a %s version", tool)
		}
		if _, err := ParseConstraint(selected, version); err != nil {
			return "", "", err
		}
		return selected, version, nil
	}
	sources := map[string]string{}
	add := func(tool, version, source string) error {
		version = strings.TrimSpace(version)
		if version == "" {
			_, selected, err := choose(tool, source+" has no version; specify tools."+tool+" in myenv.yaml")
			if err != nil {
				return err
			}
			version = selected
		}
		if _, err := ParseConstraint(tool, version); err != nil {
			return fmt.Errorf("%s: %w", source, err)
		}
		if prior := c.Tools[tool]; prior != "" && prior != version {
			merged, err := MergeConstraints(tool, prior, version)
			if err != nil {
				_, selected, inputErr := choose(tool, fmt.Sprintf("conflicting %s declarations: %s=%q, %s=%q; choose the myEnv declaration", tool, sources[tool], prior, source, version))
				if inputErr != nil {
					return inputErr
				}
				merged = selected
			}
			version = merged
		}
		c.Tools[tool] = version
		sources[tool] = source
		return nil
	}
	for _, file := range []struct{ name, tool string }{{".python-version", "python"}, {".node-version", "node"}, {".nvmrc", "node"}} {
		b, err := ReadInput(filepath.Join(dir, file.name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, false, err
		}
		if err = add(file.tool, strings.TrimPrefix(strings.TrimSpace(string(b)), "v"), file.name); err != nil {
			return nil, false, err
		}
	}
	b, err := ReadInput(filepath.Join(dir, "package.json"))
	if err == nil {
		var pkg struct {
			Engines map[string]string `json:"engines"`
		}
		if err = json.Unmarshal(b, &pkg); err != nil {
			return nil, false, fmt.Errorf("package.json: %w", err)
		}
		if v := pkg.Engines["node"]; v != "" {
			if err = add("node", v, "package.json engines.node"); err != nil {
				return nil, false, err
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, false, err
	}
	b, err = ReadInput(filepath.Join(dir, "pyproject.toml"))
	if err == nil {
		var project struct {
			Project struct {
				RequiresPython string `toml:"requires-python"`
			} `toml:"project"`
			Groups map[string][]any `toml:"dependency-groups"`
		}
		if err = toml.Unmarshal(b, &project); err != nil {
			return nil, false, fmt.Errorf("pyproject.toml: %w", err)
		}
		if v := project.Project.RequiresPython; v != "" {
			if err = add("python", v, "pyproject.toml project.requires-python"); err != nil {
				return nil, false, err
			}
		}
		c.Python = &Python{Project: "."}
		if _, ok := project.Groups["dev"]; ok {
			c.Python.Groups = []string{"dev"}
		}
		if c.Tools["python"] == "" {
			_, version, err := choose("python", "pyproject.toml has no Python version; choose a Python version")
			if err != nil {
				return nil, false, err
			}
			c.Tools["python"] = version
		}
	} else if !os.IsNotExist(err) {
		return nil, false, err
	}
	if len(c.Tools) == 0 {
		tool, version, err := choose("", "no runtime declaration found; choose python, node, java, go or rust and a version")
		if err != nil {
			return nil, false, err
		}
		c.Tools[tool] = version
	}
	encoded, err := yaml.Marshal(c)
	if err != nil {
		return nil, false, err
	}
	if _, err = Parse(encoded, dir); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		existing, e := Load(path)
		return existing, false, e
	}
	if err != nil {
		return nil, false, err
	}
	_, err = f.Write(encoded)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, false, fmt.Errorf("writing myenv.yaml failed; inspect the possibly incomplete file: %w", err)
	}
	return c, true, nil
}
