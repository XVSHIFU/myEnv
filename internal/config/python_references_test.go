package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonFileReferenceInputs(t *testing.T) {
	root := t.TempDir()
	dependency := filepath.Join(root, "local package")
	if err := os.Mkdir(dependency, 0700); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(dependency, "pyproject.toml")
	if err := os.WriteFile(local, []byte("[project]\nname='local'\nversion='1'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.ToSlash(dependency)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	reference := (&url.URL{Scheme: "file", Path: path}).String()
	for _, field := range []string{"[project]\ndependencies=", "[project.optional-dependencies]\ntest=", "[dependency-groups]\ndev="} {
		manifest := field + "['local @ " + reference + " ; python_version >= \"3.12\"']\n"
		if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest), 0600); err != nil {
			t.Fatal(err)
		}
		before, err := ReadPythonInputs(root, ".", false)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(local, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.WriteString("\n# dependency changed\n")
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		after, err := ReadPythonInputs(root, ".", false)
		if err != nil {
			t.Fatal(err)
		}
		if before.RelatedSHA256 == "" || before.RelatedSHA256 == after.RelatedSHA256 {
			t.Fatalf("file reference drift missed for %s", field)
		}
	}
}

func TestPythonFileReferenceBoundary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\ndependencies=['local @ file:../outside']\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPythonInputs(root, ".", false); err == nil {
		t.Fatal("file reference escaped workspace")
	}
	var paths []string
	if err := localPythonRequirementPaths("package @ https://example.test/file.whl", &paths); err != nil || len(paths) != 0 {
		t.Fatal("remote reference treated as local")
	}
	if err := localPythonRequirementPaths("package @ file://remote-host/share", &paths); err == nil {
		t.Fatal("remote file authority accepted as a local path")
	}
}
