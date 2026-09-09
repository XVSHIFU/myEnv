package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/core"
)

func TestStatusJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	code := Execute([]string{"-C", root, "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var got struct {
		OK, Changed bool
		Data        core.Status
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if code != 0 || !got.OK || got.Changed || got.Data.Environment != "not_ready" || got.Data.Tools["node"] != "22" || diagnostic.Len() != 0 {
		t.Fatalf("code %d output %s diagnostics %s", code, out.String(), diagnostic.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatalf("status created state: %v", err)
	}
}
