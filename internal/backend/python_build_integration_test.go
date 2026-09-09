package backend

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"myenv/internal/config"
	"myenv/internal/runner"
)

func TestPythonBuildPermissionRetained(t *testing.T) {
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
	root := t.TempDir()
	manifest := "[project]\nname='build-probe'\nversion='0.1.0'\nrequires-python='>=3.12'\n[build-system]\nrequires=[]\nbuild-backend='probe_backend'\nbackend-path=['.']\n"
	backend := `import pathlib, zipfile
pathlib.Path(__file__).with_name('build-executed').write_text('executed')
def get_requires_for_build_wheel(config_settings=None):
    return []
get_requires_for_build_editable = get_requires_for_build_wheel
def build_wheel(wheel_directory, config_settings=None, metadata_directory=None):
    name = 'build_probe-0.1.0-py3-none-any.whl'
    with zipfile.ZipFile(pathlib.Path(wheel_directory) / name, 'w') as wheel:
        wheel.writestr('build_probe.py', 'VALUE = 42\n')
        wheel.writestr('build_probe-0.1.0.dist-info/METADATA', 'Metadata-Version: 2.1\nName: build-probe\nVersion: 0.1.0\n')
        wheel.writestr('build_probe-0.1.0.dist-info/WHEEL', 'Wheel-Version: 1.0\nGenerator: myenv-test\nRoot-Is-Purelib: true\nTag: py3-none-any\n')
        wheel.writestr('build_probe-0.1.0.dist-info/RECORD', '')
    return name
build_editable = build_wheel
`
	for name, content := range map[string]string{"pyproject.toml": manifest, "probe_backend.py": backend} {
		if err = os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	u := UV{Executable: record.UV}
	r := PythonProjectRequest{Project: root, Python: record.Python, Venv: filepath.Join(root, "venv"), Cache: filepath.Join(root, "cache"), Offline: true}
	ctx := context.Background()
	err = u.SyncPythonProject(ctx, r)
	var needs *config.NeedsInput
	if !errors.As(err, &needs) {
		t.Fatalf("missing build permission not rejected: %v", err)
	}
	if _, err = os.Stat(filepath.Join(root, "build-executed")); !os.IsNotExist(err) {
		t.Fatalf("backend executed without permission: %v", err)
	}
	if err = u.CreateVenv(ctx, r.Python, r.Venv, r.Cache); err != nil {
		t.Fatal(err)
	}
	r.AllowBuild = true
	if err = u.SyncPythonProject(ctx, r); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "build-executed")); err != nil {
		t.Fatalf("authorized backend did not execute: %v", err)
	}
	entry := filepath.Join(r.Venv, "bin", "python")
	if runtime.GOOS == "windows" {
		entry = filepath.Join(r.Venv, "Scripts", "python.exe")
	}
	code, err := runner.Execute(ctx, runner.Process{Executable: entry, Args: []string{"-I", "-c", "import build_probe; assert build_probe.VALUE == 42"}, Environment: uvEnvironment(os.Environ())})
	if err != nil || code != 0 {
		t.Fatalf("built package not installed: %d %v", code, err)
	}
}
