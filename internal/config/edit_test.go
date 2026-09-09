package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetToolPreservesComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "myenv.yaml")
	if err := os.WriteFile(path, []byte("# project\nschema: 1\ntools:\n  node: \"22\" # runtime\nenv: {MODE: development}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := SetTool(path, "node", "24")
	if err != nil || !changed {
		t.Fatalf("edit %t %v", changed, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# project") || !strings.Contains(string(data), "# runtime") {
		t.Fatal("lost comments")
	}
	c, err := Load(path)
	if err != nil || c.Tools["node"] != "24" || c.Env["MODE"] != "development" {
		t.Fatalf("changed unrelated input %+v %v", c, err)
	}
	if changed, err = SetTool(path, "node", "24"); err != nil || changed {
		t.Fatalf("same version changed: %t %v", changed, err)
	}
	after, _ := os.ReadFile(path)
	if string(data) != string(after) {
		t.Fatal("no-op rewrote declaration")
	}
}
