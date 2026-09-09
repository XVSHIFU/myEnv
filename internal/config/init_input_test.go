package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitSelectedVersion(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	selectVersion := func(tool, reason string) (string, string, error) { calls++; return "node", "22", nil }
	c, changed, err := InitWithInput(dir, selectVersion)
	if err != nil || !changed || c.Tools["node"] != "22" || calls != 1 {
		t.Fatalf("result %v %v %v calls=%d", c, changed, err, calls)
	}
	_, changed, err = InitWithInput(dir, selectVersion)
	if err != nil || changed || calls != 1 {
		t.Fatal("existing configuration prompted or changed")
	}
}

func TestInitConflictSelection(t *testing.T) {
	dir := t.TempDir()
	for name, value := range map[string]string{".node-version": "22", ".nvmrc": "24"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	c, changed, err := InitWithInput(dir, func(tool, reason string) (string, string, error) {
		if tool != "node" {
			t.Fatalf("unexpected tool %s", tool)
		}
		return tool, "24", nil
	})
	if err != nil || !changed || c.Tools["node"] != "24" {
		t.Fatalf("result %v %v %v", c, changed, err)
	}
}
