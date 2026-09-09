package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
)

func TestPythonSourceBuildPermissionRetained(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained managed uv and Python")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ UV, Python string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	marker := filepath.Join(root, "dependency-build-executed")
	metadata := "Metadata-Version: 2.3\nName: sdist-probe\nVersion: 1.0.0\nRequires-Python: >=3.12\n"
	source := fmt.Sprintf(`import pathlib, zipfile
pathlib.Path(%q).write_text('executed')
def get_requires_for_build_wheel(config_settings=None):
    return []
def build_wheel(wheel_directory, config_settings=None, metadata_directory=None):
    name = 'sdist_probe-1.0.0-py3-none-any.whl'
    with zipfile.ZipFile(pathlib.Path(wheel_directory) / name, 'w') as wheel:
        wheel.writestr('sdist_probe.py', 'VALUE = 73\n')
        wheel.writestr('sdist_probe-1.0.0.dist-info/METADATA', %q)
        wheel.writestr('sdist_probe-1.0.0.dist-info/WHEEL', 'Wheel-Version: 1.0\nGenerator: myenv-test\nRoot-Is-Purelib: true\nTag: py3-none-any\n')
        wheel.writestr('sdist_probe-1.0.0.dist-info/RECORD', '')
    return name
`, marker, metadata)
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	for name, content := range map[string]string{
		"PKG-INFO":         metadata,
		"pyproject.toml":   "[project]\nname='sdist-probe'\nversion='1.0.0'\nrequires-python='>=3.12'\n[build-system]\nrequires=[]\nbuild-backend='probe_backend'\nbackend-path=['.']\n",
		"probe_backend.py": source,
	} {
		if err = tw.WriteHeader(&tar.Header{Name: "sdist_probe-1.0.0/" + name, Mode: 0600, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err = tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err = tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err = gz.Close(); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/simple/sdist-probe/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<a href="/sdist_probe-1.0.0.tar.gz#sha256=%x">sdist_probe-1.0.0.tar.gz</a>`, sha256.Sum256(archive.Bytes()))
		case "/sdist_probe-1.0.0.tar.gz":
			w.Write(archive.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	manifest := fmt.Sprintf("[project]\nname='source-consumer'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\ndependencies=['sdist-probe==1.0.0']\n[tool.uv]\npackage=false\n[[tool.uv.index]]\nurl='%s/simple'\ndefault=true\n", server.URL)
	for name, content := range map[string]string{"myenv.yaml": "schema: 1\ntools: {python: '3.12'}\npython: {project: '.'}\n", "pyproject.toml": manifest} {
		if err = os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := &Service{UV: &backend.UV{Executable: record.UV}, PythonDirectory: filepath.Dir(filepath.Dir(record.Python))}
	ctx := context.Background()
	denied, err := s.Sync(ctx, SyncRequest{Directory: root})
	if _, markerErr := os.Stat(marker); !os.IsNotExist(markerErr) {
		t.Fatal("source backend executed without permission")
	}
	var needs *config.NeedsInput
	if !errors.As(err, &needs) || denied.Changed {
		t.Fatalf("source permission %+v %v", denied, err)
	}
	changed, err := s.Sync(ctx, SyncRequest{Directory: root, ConfirmBuild: func(string) (bool, error) {
		return true, os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest+"\n# changed during confirmation\n"), 0600)
	}})
	if err == nil || !strings.HasPrefix(err.Error(), "INPUT_CHANGED:") || changed.Changed {
		t.Fatalf("confirmation input change %+v %v", changed, err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("build executed after input changed during confirmation")
	}
	if err = os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	confirmations := 0
	applied, err := s.Sync(ctx, SyncRequest{Directory: root, ConfirmBuild: func(string) (bool, error) { confirmations++; return true, nil }})
	if err != nil || !applied.Changed || confirmations != 1 {
		t.Fatalf("source approval %+v %v confirmations=%d", applied, err, confirmations)
	}
	if _, err = os.Stat(marker); err != nil {
		t.Fatal("authorized source did not build")
	}
	var output bytes.Buffer
	code, err := runner.Execute(ctx, runner.Process{Executable: applied.Generation.PythonExecutable, Args: []string{"-I", "-c", "import sdist_probe; assert sdist_probe.VALUE == 73"}, Environment: os.Environ(), Directory: root, Stdout: &output, Stderr: &output})
	if err != nil || code != 0 {
		t.Fatalf("source install %d %v %s", code, err, output.String())
	}
	// A fresh --locked installation has no previously built wheel cache.
	lockedRoot := t.TempDir()
	for _, name := range []string{"myenv.yaml", "myenv.lock", "pyproject.toml", "uv.lock"} {
		contents, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(lockedRoot, name), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	denied, err = s.Sync(ctx, SyncRequest{Directory: lockedRoot, Locked: true})
	if !errors.As(err, &needs) || denied.Changed || denied.LockChanged || denied.NativeLockChanged {
		t.Fatalf("locked source permission %+v %v", denied, err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("locked source backend executed without permission")
	}
	confirmations = 0
	applied, err = s.Sync(ctx, SyncRequest{Directory: lockedRoot, Locked: true, ConfirmBuild: func(string) (bool, error) { confirmations++; return true, nil }})
	if err != nil || !applied.Changed || applied.LockChanged || applied.NativeLockChanged || confirmations != 1 {
		t.Fatalf("locked source approval %+v %v confirmations=%d", applied, err, confirmations)
	}
	for _, name := range []string{"myenv.lock", "uv.lock"} {
		before, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(filepath.Join(lockedRoot, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("locked build modified %s", name)
		}
	}
}
