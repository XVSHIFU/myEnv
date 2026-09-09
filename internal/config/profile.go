package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// ProfilePath resolves only the current user's configuration namespace. An
// explicit directory supports isolated integrations; it is not a project cwd.
// Resolution is read-only and does not initialize configuration or state.
func ProfilePath(userConfigDirectory string) (string, error) {
	if userConfigDirectory == "" {
		var err error
		userConfigDirectory, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	if !filepath.IsAbs(userConfigDirectory) {
		return "", fmt.Errorf("user configuration directory must be absolute")
	}
	return filepath.Join(userConfigDirectory, "myenv", "profile.yaml"), nil
}

func LoadProfile(path string) (*Config, error) {
	data, err := ReadInput(path)
	if err != nil {
		return nil, err
	}
	return ParseProfile(data, filepath.Dir(path))
}

// ParseProfile shares strict YAML and runtime validation with project config,
// but does not admit project paths, dependency groups or project environment.
func ParseProfile(data []byte, root string) (*Config, error) {
	c, err := parseConfig(data, root, true)
	if err != nil {
		return nil, err
	}
	var profile struct {
		Schema int               `yaml:"schema"`
		Tools  map[string]string `yaml:"tools"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err = decoder.Decode(&profile); err != nil {
		return nil, fmt.Errorf("invalid user profile: %w", err)
	}
	return c, nil
}

// SetProfileTool requires the profile modification lock. First use creates an
// exclusive declaration; existing profiles retain comments and other tools.
func SetProfileTool(path, tool, version string) (bool, error) {
	if _, err := ParseConstraint(tool, version); err != nil {
		return false, err
	}
	if _, err := os.Lstat(path); err == nil {
		return setTool(path, tool, version, ParseProfile)
	} else if !os.IsNotExist(err) {
		return false, err
	}
	data, err := yaml.Marshal(&Config{Schema: 1, Tools: map[string]string{tool: version}})
	if err != nil {
		return false, err
	}
	if _, err = ParseProfile(data, filepath.Dir(path)); err != nil {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false, err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return true, err
	}
	return true, closeErr
}
