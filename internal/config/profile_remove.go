package config

import (
	"bytes"
	"fmt"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
)

// RemoveProfileTool edits only the declaration under the caller's workspace lock.
func RemoveProfileTool(path, tool string) (bool, error) {
	original, err := ReadInput(path)
	if err != nil {
		return false, err
	}
	c, err := ParseProfile(original, filepath.Dir(path))
	if err != nil {
		return false, err
	}
	if _, ok := c.Tools[tool]; !ok {
		return false, nil
	}
	var document yaml.Node
	if err = yaml.Unmarshal(original, &document); err != nil {
		return false, err
	}
	root := document.Content[0]
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value != "tools" {
			continue
		}
		tools := root.Content[i+1]
		for j := 0; j < len(tools.Content); j += 2 {
			if tools.Content[j].Value == tool {
				tools.Content = append(tools.Content[:j], tools.Content[j+2:]...)
				break
			}
		}
	}
	data, err := yaml.Marshal(&document)
	if err != nil {
		return false, err
	}
	if _, err = ParseProfile(data, filepath.Dir(path)); err != nil {
		return false, err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".myenv-remove-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
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
		return false, fmt.Errorf("INPUT_CHANGED: user profile changed during removal")
	}
	if err = replaceFile(file.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}
