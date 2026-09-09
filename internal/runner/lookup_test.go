package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupChildPath(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(bin, "tool.EXE")
	if err := os.WriteFile(file, []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := Lookup("tool", root, []string{"Path=" + bin, "PATHEXT=.EXE"}, true)
	if err != nil || got != file {
		t.Fatalf("lookup %s %v", got, err)
	}
	if _, err = Lookup("tool", root, []string{"PATH="}, true); err == nil {
		t.Fatal("leaked parent PATH")
	}
	got, err = Lookup("./bin/tool.EXE", root, nil, true)
	if err != nil || got != file {
		t.Fatalf("relative lookup %s %v", got, err)
	}
}
