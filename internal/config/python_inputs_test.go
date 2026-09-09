package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonInputDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname='sample'\nversion='1'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	initial, err := ReadPythonInputs(root, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if initial.UVLockSHA256 != "" {
		t.Fatal("missing lock represented as present")
	}
	if initial.ConfigPolicy != "project-only-v1" {
		t.Fatal("missing explicit configuration policy")
	}
	legacy := initial
	legacy.ConfigPolicy = ""
	if legacy.Digest() == initial.Digest() {
		t.Fatal("configuration policy did not affect input digest")
	}
	if _, err = ReadPythonInputs(root, ".", true); err == nil || !strings.HasPrefix(err.Error(), "LOCK_OUT_OF_DATE:") {
		t.Fatalf("locked accepted missing lock: %v", err)
	}
	prior := initial.Digest()
	for _, file := range []string{"uv.lock", "uv.toml", "pyproject.toml"} {
		contents := "changed"
		if file == "pyproject.toml" {
			contents = "[project]\nname='changed'\nversion='1'\n"
		}
		if err = os.WriteFile(filepath.Join(root, file), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		next, err := ReadPythonInputs(root, ".", false)
		if err != nil {
			t.Fatal(err)
		}
		if next.Digest() == prior {
			t.Fatalf("missed %s change", file)
		}
		prior = next.Digest()
	}
	if _, err = ReadPythonInputs(root, "../outside", false); err == nil {
		t.Fatal("accepted external project")
	}
}
