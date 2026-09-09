package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnvironmentCaseConflict(t *testing.T) {
	input := []byte("schema: 1\ntools: {node: '22'}\nenv: {Path: first-private-value, PATH: second-private-value}\n")
	_, err := Parse(input, t.TempDir())
	if runtime.GOOS == "windows" {
		if err == nil || err.Error() != `duplicate case-insensitive environment key "PATH"` {
			t.Fatalf("Windows conflict diagnostic: %v", err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
}

func TestParseBoundaries(t *testing.T) {
	root := t.TempDir()
	valid := "schema: 1\ntools:\n  python: \"3.12\"\npython:\n  project: .\nenv:\n  APP_ENV: development\n"
	if _, err := Parse([]byte(valid), root); err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string]string{
		"duplicate": "schema: 1\nschema: 1\ntools: {node: \"22\"}",
		"unknown":   "schema: 1\ntools: {node: \"22\"}\nother: x",
		"number":    "schema: 1\ntools: {node: 22}",
		"tag":       "schema: 1\ntools: {node: !danger x}",
		"alias":     "schema: 1\ntools: &x {node: \"22\"}\nenv: *x",
		"escape":    "schema: 1\ntools: {python: \"3.12\"}\npython: {project: ../outside}",
		"multiple":  "schema: 1\ntools: {node: \"22\"}\n---\nschema: 1",
		"empty":     "",
		"large":     strings.Repeat(" ", MaxInputSize+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(input), root); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}

func TestDiscoverVCSBoundary(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "repo", "child")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "repo", ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(nested); !errors.Is(err, ErrNoProject) {
		t.Fatalf("crossed VCS boundary: %v", err)
	}
	project := filepath.Join(root, "repo")
	if err := os.WriteFile(filepath.Join(project, "myenv.yaml"), []byte("schema: 1"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := Discover(nested); err != nil || got != project {
		t.Fatalf("got %q: %v", got, err)
	}
}
