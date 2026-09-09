package backend

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestPythonManagedPrepare(t *testing.T) {
	record := os.Getenv("MYENV_TEST_UV_PREPARED")
	if record == "" {
		t.Skip("explicit managed Python preparation gate")
	}
	root := os.Getenv("MYENV_TEST_ARTIFACTS")
	version := os.Getenv("MYENV_TEST_PYTHON_VERSION")
	if !filepath.IsAbs(root) || version == "" {
		t.Fatal("isolated artifact directory and exact Python version required")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var u UV
	if err = json.Unmarshal(data, &u); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	python, err := u.InstallPython(ctx, version, filepath.Join(root, "runtimes"), filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	venv := filepath.Join(root, "venv-"+time.Now().Format("20060102T150405.000000000"))
	if err = u.CreateVenv(ctx, python, venv, filepath.Join(root, "cache")); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(venv, "bin", "python")
	if runtime.GOOS == "windows" {
		entry = filepath.Join(venv, "Scripts", "python.exe")
	}
	if err = verifyPython(ctx, entry, version); err != nil {
		t.Fatal(err)
	}
	prepared := struct{ UV, Python, Venv, Executable, Version string }{u.Executable, python, venv, entry, version}
	data, err = json.MarshalIndent(prepared, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "python-prepared.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("verified managed Python %s and final venv at %s", version, venv)
}

func TestPythonRetainedResolve(t *testing.T) {
	record := os.Getenv("MYENV_TEST_UV_PREPARED")
	if record == "" {
		t.Skip("requires retained managed uv")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var u UV
	if err = json.Unmarshal(data, &u); err != nil {
		t.Fatal(err)
	}
	for _, selector := range []string{"3.12", ">=3.12.12,<3.13"} {
		release, err := u.ResolvePython(context.Background(), selector, filepath.Join(t.TempDir(), "cache"))
		if err != nil || release.Version != "3.12.13" || release.Evidence != "version" {
			t.Fatalf("resolve %s: %+v %v", selector, release, err)
		}
	}
}

func TestRetainedVenvIdentity(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained managed Python and venv")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ Python, Venv, Executable string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = VerifyVenv(ctx, record.Executable, record.Venv); err != nil {
		t.Fatal(err)
	}
	if err = VerifyVenv(ctx, record.Python, record.Venv); err == nil {
		t.Fatal("base Python accepted as project venv")
	}
	if err = VerifyVenv(ctx, record.Executable, t.TempDir()); err == nil {
		t.Fatal("accepted different venv destination")
	}
}

func TestPythonProjectRetainedSync(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained managed Python")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ UV, Python string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	if err = os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(parent, "uv.toml"), []byte("invalid parent configuration ["), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(parent, "pyproject.toml"), []byte("[tool.uv.workspace]\nmembers=[]\nexclude=['project']\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// No tool.uv table: native automatic discovery would reach the parent.
	manifest := "[project]\nname='myenv-test'\nversion='0.1.0'\nrequires-python='>=3.12'\ndependencies=[]\n[dependency-groups]\ndev=[]\n"
	if err = os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	u := UV{Executable: record.UV}
	r := PythonProjectRequest{Project: root, Python: record.Python, Venv: filepath.Join(root, "venv"), Cache: filepath.Join(root, "cache"), Groups: []string{"dev"}, Offline: true}
	ctx := context.Background()
	if err = u.CreateVenv(ctx, r.Python, r.Venv, r.Cache); err != nil {
		t.Fatal(err)
	}
	if err = u.SyncPythonProject(ctx, r); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "uv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	r.Locked = true
	if err = u.SyncPythonProject(ctx, r); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "uv.lock"))
	if err != nil || string(after) != string(before) {
		t.Fatal("locked sync changed native lock")
	}
	if err = os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest+"\n[build-system]\nrequires=[]\nbuild-backend='dangerous'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = u.SyncPythonProject(ctx, r); err == nil {
		t.Fatal("accepted first-party build without permission")
	}
}
