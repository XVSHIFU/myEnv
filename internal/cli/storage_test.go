package cli

import (
	"myenv/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeServiceStorage(t *testing.T) {
	want, err := config.ResolveUserStorage("", "")
	if err != nil {
		t.Fatal(err)
	}
	s, err := runtimeService(false, "")
	if err != nil || s.Storage == nil || *s.Storage != want {
		t.Fatalf("default storage %+v %v", s, err)
	}
	namespace := t.TempDir()
	a, err := runtimeService(false, namespace)
	if err != nil {
		t.Fatal(err)
	}
	b, err := runtimeService(true, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if *a.Storage != *b.Storage || a.Profile || !b.Profile {
		t.Fatal("project/profile storage is not shared")
	}
	for _, path := range []string{a.Storage.Data, a.Storage.Cache} {
		relative, err := filepath.Rel(namespace, path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = config.Within(namespace, relative); err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("service construction created storage", err)
		}
	}
	if _, err = runtimeService(false, "relative"); err == nil {
		t.Fatal("relative injected namespace accepted")
	}
}
