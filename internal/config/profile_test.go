package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileIsolationAndValidation(t *testing.T) {
	root := t.TempDir()
	profile, err := ProfilePath(root)
	if err != nil || profile != filepath.Join(root, "myenv", "profile.yaml") {
		t.Fatalf("profile location %q %v", profile, err)
	}
	if _, err = os.Stat(filepath.Dir(profile)); !os.IsNotExist(err) {
		t.Fatal("path resolution initialized a profile")
	}
	if err = os.MkdirAll(filepath.Dir(profile), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(profile, []byte("schema: 1\ntools: {node: '22', python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(filepath.Dir(profile), "myenv.yaml"), []byte("schema: 1\ntools: {node: '24'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadProfile(profile)
	if err != nil || c.Tools["node"] != "22" || c.Tools["python"] != "3.12" {
		t.Fatalf("profile read %+v %v", c, err)
	}
	for _, extra := range []string{"python: null\n", "env: {}\n", "env: null\n", "unknown: true\n", "tools: {node: '24'}\n"} {
		if _, err := ParseProfile([]byte("schema: 1\ntools: {node: '22'}\n"+extra), root); err == nil {
			t.Fatalf("accepted profile field %q", extra)
		}
	}
	if _, err = ProfilePath("relative"); err == nil {
		t.Fatal("accepted a cwd-relative user namespace")
	}
}
