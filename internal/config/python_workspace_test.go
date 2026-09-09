package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonWorkspaceInputs(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("pyproject.toml", "[tool.uv.workspace]\nmembers=['packages/*', 'nested/**/pkg']\nexclude=['packages/skip', 'nested/skip/**']\n")
	for _, path := range []string{"packages/a", "packages/skip", "nested/a/b/pkg", "nested/skip/pkg"} {
		write(path+"/pyproject.toml", "[project]\nname='fixture'\nversion='1'\n")
	}
	read := func() PythonInputs {
		t.Helper()
		inputs, err := ReadPythonInputs(root, ".", false)
		if err != nil {
			t.Fatal(err)
		}
		return inputs
	}
	initial := read()
	if initial.RelatedSHA256 == "" {
		t.Fatal("workspace member metadata absent")
	}
	write("nested/skip/pkg/pyproject.toml", "excluded metadata change")
	write("packages/skip/pyproject.toml", "excluded metadata change")
	if after := read(); after != initial {
		t.Fatal("excluded member affected inputs")
	}
	write("nested/a/b/pkg/pyproject.toml", "[project]\nname='fixture'\nversion='2'\n")
	changed := read()
	if changed.RelatedSHA256 == initial.RelatedSHA256 || changed.PyprojectSHA256 != initial.PyprojectSHA256 {
		t.Fatal("recursive member change not detected")
	}
	write("packages/new/pyproject.toml", "[project]\nname='new'\nversion='1'\n")
	if added := read(); added.RelatedSHA256 == changed.RelatedSHA256 {
		t.Fatal("new glob member not detected")
	}
}

func TestPythonWorkspacePatternBoundaries(t *testing.T) {
	for _, pattern := range []string{"../external", "packages/../../external", "[broken"} {
		if _, err := workspacePatterns([]any{pattern}); err == nil {
			t.Fatalf("accepted %q", pattern)
		}
	}
	root := t.TempDir()
	for _, name := range []string{"one", "two", "three"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	budget := 2
	_, err := pythonWorkspaceMembers(root, ".", map[string]any{"members": []any{"*"}}, &budget)
	if err == nil || !strings.Contains(err.Error(), "discovery exceeds") {
		t.Fatalf("ignored discovery budget: %v", err)
	}
}
