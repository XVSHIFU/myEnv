package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonBuildNoninteractive(t *testing.T) {
	root := t.TempDir()
	for name, value := range map[string]string{
		"myenv.yaml":     "schema: 1\ntools: {python: '3.12'}\npython: {project: '.'}\n",
		"pyproject.toml": "[project]\nname='myenv-build-test'\nversion='0.1.0'\n[build-system]\nrequires=[]\nbuild-backend='must_not_execute'\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var output, diagnostic bytes.Buffer
	// Available input must not constitute approval in JSON/noninteractive mode.
	code := Execute([]string{"-C", root, "sync", "--json", "--no-input"}, bytes.NewBufferString("yes\n"), &output, &diagnostic, "test")
	var response result
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if code != 3 || response.OK || response.Changed || response.Error == nil || response.Error.Code != "NEEDS_INPUT" {
		t.Fatalf("code %d %s %s", code, output.String(), diagnostic.String())
	}
	for _, name := range []string{"myenv.lock", "uv.lock", filepath.Join(".myenv", "backends"), filepath.Join(".myenv", "generations")} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("preflight prepared %s: %v", name, err)
		}
	}
}
