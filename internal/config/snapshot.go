package config

import (
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
)

func WriteSnapshot(directory string, c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(directory, "config.yaml"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func ReadSnapshot(directory, workspace string) (*Config, error) {
	data, err := ReadInput(filepath.Join(directory, "config.yaml"))
	if err != nil {
		return nil, err
	}
	c, e := Parse(data, workspace)
	if e != nil {
		return ParseProfile(data, workspace)
	}
	return c, nil
}
