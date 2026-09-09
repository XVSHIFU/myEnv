package core

import (
	"context"
	"io"
	"myenv/internal/backend"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExternalUVRoundTrip(t *testing.T) {
	root := os.Getenv("MYENV_EXTERNAL_TEST_ROOT")
	manager := os.Getenv("MYENV_EXTERNAL_TEST_UV")
	if root == "" || manager == "" {
		t.Skip("explicit isolated external-manager trial required")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(manager) {
		t.Fatal("absolute paths required")
	}
	t.Setenv("UV_PYTHON_INSTALL_DIR", filepath.Join(root, "installs"))
	t.Setenv("UV_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("UV_PYTHON_INSTALL_BIN", "0")
	t.Setenv("UV_PYTHON_INSTALL_REGISTRY", "0")
	os.MkdirAll(root, 0700)
	marker := filepath.Join(root, "keep.txt")
	os.WriteFile(marker, []byte("preserve"), 0600)
	python, e := (backend.UV{Executable: manager}).InstallPython(context.Background(), "3.12.13", filepath.Join(root, "installs"), filepath.Join(root, "cache"))
	if e != nil {
		t.Fatal(e)
	}
	row := Installation{Owner: "external", Tool: "python", Path: python}
	if data, e := managerRead(context.Background(), manager, filepath.Dir(manager), "python", "list", "--only-installed", "--managed-python", "--output-format", "json", "--no-config", "--offline", "--no-python-downloads"); e == nil {
		t.Logf("selected %s; manager directory %s; catalog %s", python, filepath.Dir(manager), data)
	}
	for _, action := range []string{"repair", "remove"} {
		plan, e := BuildExternalPlan(context.Background(), row, manager, action)
		if e != nil {
			t.Fatal(e)
		}
		if e = ApplyExternalPlan(context.Background(), plan, nil, io.Discard, io.Discard); e != nil {
			t.Fatal(e)
		}
		if action == "repair" {
			if _, e = os.Stat(python); e != nil {
				t.Fatal(e)
			}
		}
	}
	if _, e = os.Stat(python); !os.IsNotExist(e) {
		t.Fatalf("runtime still exists: %v (%s)", e, runtime.GOOS)
	}
	if _, e = os.Stat(marker); e != nil {
		t.Fatal("removed outside target", e)
	}
}
