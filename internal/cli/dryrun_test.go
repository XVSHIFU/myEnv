package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDryRunLockedMissingLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	code := Execute([]string{"-C", root, "sync", "--dry-run", "--locked", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var got result
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if code != 1 || got.OK || got.Changed || got.Error == nil || got.Error.Code != "LOCK_OUT_OF_DATE" {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatalf("preview created state: %v", err)
	}
}
