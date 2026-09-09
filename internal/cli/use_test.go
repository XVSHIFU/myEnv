package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/config"
	"myenv/internal/core"
)

func TestUseMissingSelectionNonInteractive(t *testing.T) {
	for _, flags := range [][]string{nil, {"--no-input"}, {"--json"}} {
		root := t.TempDir()
		input := bytes.NewBufferString("node@22\n")
		var out, diagnostic bytes.Buffer
		args := append([]string{"-C", root, "use"}, flags...)
		if code := execute(args, input, &out, &diagnostic, "test", root); code != 2 {
			t.Fatalf("%v: exit %d: %s %s", flags, code, &out, &diagnostic)
		}
		if input.Len() != len("node@22\n") {
			t.Fatal("non-interactive use consumed input")
		}
		entries, err := os.ReadDir(root)
		if err != nil || len(entries) != 0 {
			t.Fatalf("missing selection created state: %v %v", entries, err)
		}
	}
}

func TestUseFailureReportsDeclarationChange(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: \"22\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// Invalid lock fails sync before any network request, after use edits YAML.
	if err := os.WriteFile(filepath.Join(root, "myenv.lock"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	code := Execute([]string{"-C", root, "use", "node@24", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var response struct {
		OK, Changed bool
		Data        core.UseResult
		Error       *failure
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if code == 0 || response.OK || !response.Changed || !response.Data.DeclarationChanged || response.Error == nil {
		t.Fatalf("failure not reported accurately: %d %s", code, out.String())
	}
	c, err := config.Load(filepath.Join(root, "myenv.yaml"))
	if err != nil || c.Tools["node"] != "24" {
		t.Fatalf("declaration %+v %v", c, err)
	}
}
