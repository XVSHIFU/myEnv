package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// SetTool requires the caller's workspace modification lock. It preserves YAML
// comments and unrelated values, but may normalize whitespace.
func SetTool(path, tool, version string) (bool, error) {
	return setTool(path, tool, version, Parse)
}

func setTool(path, tool, version string, parse func([]byte, string) (*Config, error)) (bool, error) {
	if _, err := ParseConstraint(tool, version); err != nil {
		return false, err
	}
	original, err := ReadInput(path)
	if err != nil {
		return false, err
	}
	c, err := parse(original, filepath.Dir(path))
	if err != nil {
		return false, err
	}
	if c.Tools[tool] == version {
		return false, nil
	}
	var document yaml.Node
	if err = yaml.Unmarshal(original, &document); err != nil {
		return false, err
	}
	root := document.Content[0]
	var tools *yaml.Node
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "tools" {
			tools = root.Content[i+1]
			break
		}
	}
	if tools == nil {
		return false, fmt.Errorf("configuration has no tools mapping")
	}
	found := false
	for i := 0; i < len(tools.Content); i += 2 {
		if tools.Content[i].Value == tool {
			tools.Content[i+1].Value = version
			tools.Content[i+1].Tag = "!!str"
			tools.Content[i+1].Style = yaml.DoubleQuotedStyle
			found = true
			break
		}
	}
	if !found {
		tools.Content = append(tools.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: tool}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: version, Style: yaml.DoubleQuotedStyle})
	}
	data, err := yaml.Marshal(&document)
	if err != nil {
		return false, err
	}
	if _, err = parse(data, filepath.Dir(path)); err != nil {
		return false, err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".myenv-config-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return false, err
	}
	if closeErr != nil {
		return false, closeErr
	}
	latest, err := ReadInput(path)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(latest, original) {
		return false, fmt.Errorf("INPUT_CHANGED: declaration changed while editing")
	}
	if err = replaceFile(f.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}
