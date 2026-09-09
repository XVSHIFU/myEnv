package backend

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestUVEnvironmentIsolation(t *testing.T) {
	parent := []string{"PATH=system", "UV_VENV_CLEAR=1", "uv_venv_seed=1", "PYTHONPATH=external", "VIRTUAL_ENV=other", "CONDA_PREFIX=other", "HTTPS_PROXY=proxy"}
	want := []string{"PATH=system", "HTTPS_PROXY=proxy"}
	if got := uvEnvironment(parent); !reflect.DeepEqual(got, want) {
		t.Fatalf("environment %v", got)
	}
	if len(parent) != 7 {
		t.Fatal("mutated parent environment")
	}
}

func TestFixedUVReleases(t *testing.T) {
	for _, platform := range []string{"windows-amd64", "linux-amd64-glibc", "darwin-arm64"} {
		release, err := FixedUVRelease(platform)
		if err != nil {
			t.Fatal(err)
		}
		digest, err := hex.DecodeString(release.SHA256)
		if err != nil || len(digest) != 32 || release.Version != UVVersion || !strings.Contains(release.URL, "/"+UVVersion+"/") {
			t.Fatalf("invalid fixed release %+v %v", release, err)
		}
	}
	if _, err := FixedUVRelease("linux-musl"); err == nil {
		t.Fatal("accepted unsupported release")
	}
}

func TestUVRefusesExistingVenv(t *testing.T) {
	root := t.TempDir()
	// Invalid engine ensures an existing destination is rejected before launch.
	u := UV{Executable: filepath.Join(root, "missing-uv")}
	if err := u.CreateVenv(context.Background(), filepath.Join(root, "python"), root, filepath.Join(root, "cache")); err == nil || err.Error() != "venv destination must not exist" {
		t.Fatalf("existing directory not protected: %v", err)
	}
}

func TestPythonInstallRejectsUnresolvedRequests(t *testing.T) {
	root := t.TempDir()
	u := UV{Executable: filepath.Join(root, "missing-uv")}
	for _, version := range []string{"3.12", ">=3.12", "latest", "-r", "3.12.1/path"} {
		if _, err := u.InstallPython(context.Background(), version, filepath.Join(root, "python"), filepath.Join(root, "cache")); err == nil || err.Error() != "Python installation requires an exact three-part version" {
			t.Fatalf("request %q: %v", version, err)
		}
	}
}
