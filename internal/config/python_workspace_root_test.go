package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonWorkspaceSharedInputs(t *testing.T) {
	root := t.TempDir()
	write := func(name, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("pyproject.toml", "[tool.uv.workspace]\nmembers=['packages/*']\nexclude=['packages/independent']\n")
	write("uv.lock", "shared lock bytes")
	write("uv.toml", "no-index=true\n")
	for _, member := range []string{"a", "b", "independent"} {
		write("packages/"+member+"/pyproject.toml", "[project]\nname='"+member+"'\nversion='1'\n")
	}
	first, err := ReadPythonInputs(root, "packages/a", true)
	if err != nil || first.WorkspaceRoot != "." || first.UVLockSHA256 == "" || first.WorkspaceSHA256 == "" {
		t.Fatalf("shared inputs %+v %v", first, err)
	}
	write("packages/b/pyproject.toml", "[project]\nname='b'\nversion='2'\n")
	second, err := ReadPythonInputs(root, "packages/a", true)
	if err != nil || second.RelatedSHA256 == first.RelatedSHA256 || second.PyprojectSHA256 != first.PyprojectSHA256 {
		t.Fatalf("sibling change %+v %v", second, err)
	}
	write("pyproject.toml", "[tool.uv.workspace]\nmembers=['packages/*']\nexclude=['packages/independent']\n# root change\n")
	third, err := ReadPythonInputs(root, "packages/a", true)
	if err != nil || third.WorkspaceSHA256 == second.WorkspaceSHA256 {
		t.Fatalf("root change %+v %v", third, err)
	}
	independent, err := ReadPythonInputs(root, "packages/independent", false)
	if err != nil || independent.WorkspaceRoot != "" || independent.UVLockSHA256 != "" || independent.UVConfigSHA256 != "" {
		t.Fatalf("excluded project inherited workspace: %+v %v", independent, err)
	}
	if _, err := ReadPythonInputs(filepath.Join(root, "packages", "a"), ".", false); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("adopted parent outside myEnv boundary: %v", err)
	}
}
