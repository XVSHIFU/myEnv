package core

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInventoryExternalReadOnly(t *testing.T) {
	root := t.TempDir()
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("not executed by shallow inventory"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+root)
	s := Service{UserConfigDirectory: filepath.Join(root, "namespace")}
	rows, err := s.Inventory(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, row := range rows.Installations {
		if row.Path == path {
			found++
			if row.Owner != "external" || len(row.Actions) != 1 || row.Actions[0] != "inspect" || row.State != "unverified" {
				t.Fatalf("claimed unsupported ownership/health: %+v", row)
			}
		}
	}
	if found != 1 {
		t.Fatalf("expected one deduplicated external install, got %d", found)
	}
}
func TestInventoryMissingDeclaredHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("JAVA_HOME", filepath.Join(root, "missing"))
	s := Service{UserConfigDirectory: filepath.Join(root, "namespace")}
	rows, err := s.Inventory(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows.Installations {
		if row.Source == "JAVA_HOME" {
			if row.State != "broken" {
				t.Fatal(row)
			}
			return
		}
	}
	t.Fatal("missing JAVA_HOME omitted")
}
