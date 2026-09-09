package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func MarkComplete(directory, digest string) error {
	file, err := os.OpenFile(filepath.Join(directory, "complete"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = file.WriteString(digest + "\n")
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func CheckComplete(directory, digest string) error {
	data, err := ReadInput(filepath.Join(directory, "complete"))
	if err != nil {
		return err
	}
	if string(data) != digest+"\n" {
		return fmt.Errorf("generation completion marker differs from input digest")
	}
	return nil
}
