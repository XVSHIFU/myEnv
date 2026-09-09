package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonRelatedInputs(t *testing.T) {
	root := t.TempDir()
	write := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("pyproject.toml", "[project]\nname='consumer'\nversion='1'\n[tool.uv.sources]\na={path='packages/a'}\n")
	write("packages/a/pyproject.toml", "[project]\nname='a'\nversion='1'\n[tool.uv.sources]\nb={path='../b'}\n")
	write("packages/b/pyproject.toml", "[project]\nname='b'\nversion='1'\n[tool.uv.sources]\na={path='../a'}\n")
	read := func() PythonInputs {
		t.Helper()
		p, err := ReadPythonInputs(root, ".", false)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	before := read()
	if before.RelatedSHA256 == "" {
		t.Fatal("local source metadata missing")
	}
	write("packages/b/setup.cfg", "[metadata]\nname=b\n")
	after := read()
	if before.RelatedSHA256 == after.RelatedSHA256 || before.PyprojectSHA256 != after.PyprojectSHA256 {
		t.Fatal("transitive source metadata change was not recorded independently")
	}
	if repeat := read(); repeat != after {
		t.Fatal("source cycle produced unstable digest")
	}
	write("packages/a/source.py", "print('business code, not metadata')")
	if repeat := read(); repeat != after {
		t.Fatal("scanned unrelated source code")
	}
	write("pyproject.toml", "[project]\nname='consumer'\nversion='1'\n[tool.uv.sources]\na={path='../outside'}\n")
	if _, err := ReadPythonInputs(root, ".", false); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("accepted external source: %v", err)
	}
}

func TestPythonLocalArchiveDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.uv.sources]\na={path='a.whl'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "a.whl")
	if err := os.WriteFile(archive, []byte("first test archive"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := ReadPythonInputs(root, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("second test archive"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := ReadPythonInputs(root, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if before.RelatedSHA256 == after.RelatedSHA256 {
		t.Fatal("local artifact content changed without input drift")
	}
}
