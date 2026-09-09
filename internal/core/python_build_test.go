package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
)

func TestPythonCoreBuildPermissionRetained(t *testing.T) {
	testPythonCoreBuildPermission(t, "root")
}

func TestPythonLocalSourceRetained(t *testing.T) {
	testPythonCoreBuildPermission(t, "local")
}

func TestPythonWorkspaceSourceRetained(t *testing.T) {
	testPythonCoreBuildPermission(t, "workspace")
}

func TestPythonWorkspaceMemberRetained(t *testing.T) {
	testPythonCoreBuildPermission(t, "member")
}

func TestPythonFileSourceRetained(t *testing.T) {
	testPythonCoreBuildPermission(t, "file")
}

func testPythonCoreBuildPermission(t *testing.T, mode string) {
	local := mode != "root"
	workspace := mode == "workspace" || mode == "member"
	member := mode == "member"
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained managed Python and uv")
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
	source := `import pathlib, zipfile
pathlib.Path(__file__).with_name('build-executed').write_text('executed')
def get_requires_for_build_wheel(config_settings=None):
    return []
get_requires_for_build_editable = get_requires_for_build_wheel
def build_wheel(wheel_directory, config_settings=None, metadata_directory=None):
    name = 'core_probe-0.1.0-py3-none-any.whl'
    with zipfile.ZipFile(pathlib.Path(wheel_directory) / name, 'w') as wheel:
        wheel.writestr('core_probe.py', 'VALUE = 42\n')
        wheel.writestr('core_probe-0.1.0.dist-info/METADATA', 'Metadata-Version: 2.1\nName: core-probe\nVersion: 0.1.0\n')
        wheel.writestr('core_probe-0.1.0.dist-info/WHEEL', 'Wheel-Version: 1.0\nGenerator: myenv-test\nRoot-Is-Purelib: true\nTag: py3-none-any\n')
        wheel.writestr('core_probe-0.1.0.dist-info/RECORD', '')
    return name
build_editable = build_wheel
`
	for name, value := range map[string]string{
		"myenv.yaml":      "schema: 1\ntools: {python: '3.12'}\npython: {project: '.'}\n",
		"pyproject.toml":  "[project]\nname='core-probe'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\n[build-system]\nrequires=[]\nbuild-backend='core_backend'\nbackend-path=['.']\n",
		"core_backend.py": source,
	} {
		if err = os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	packageRoot := root
	if local {
		packageRoot = filepath.Join(root, "dependency")
		if mode == "file" {
			packageRoot = filepath.Join(root, "local dependency")
		}
		if err = os.Mkdir(packageRoot, 0700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"pyproject.toml", "core_backend.py"} {
			if err = os.Rename(filepath.Join(root, name), filepath.Join(packageRoot, name)); err != nil {
				t.Fatal(err)
			}
		}
		consumer := "[project]\nname='consumer'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\ndependencies=['core-probe']\n[tool.uv.sources]\ncore-probe={path='dependency'}\n"
		if workspace {
			consumer = strings.Replace(consumer, "path='dependency'", "workspace=true", 1) + "[tool.uv.workspace]\nmembers=['dependency']\n"
		}
		if mode == "file" {
			path := filepath.ToSlash(packageRoot)
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			reference := (&url.URL{Scheme: "file", Path: path}).String()
			consumer = "[project]\nname='consumer'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\ndependencies=['core-probe @ " + reference + "']\n"
		}
		if err = os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(consumer), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if member {
		if err = os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\npython: {project: 'dependency'}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(packageRoot, "uv.toml"), []byte("ignored member config ["), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := &Service{UV: &backend.UV{Executable: record.UV}, PythonDirectory: filepath.Dir(filepath.Dir(record.Python))}
	ctx := context.Background()
	confirmations := 0
	request := SyncRequest{Directory: root, ConfirmBuild: func(string) (bool, error) { confirmations++; return false, nil }}
	denied, err := s.Sync(ctx, request)
	var needs *config.NeedsInput
	if !errors.As(err, &needs) || denied.Changed || confirmations != 1 {
		t.Fatalf("denied %+v %v confirmations=%d", denied, err, confirmations)
	}
	if _, err = os.Stat(filepath.Join(packageRoot, "build-executed")); !os.IsNotExist(err) {
		t.Fatal("declined build executed")
	}
	confirmations = 0
	request.ConfirmBuild = func(string) (bool, error) { confirmations++; return true, nil }
	applied, err := s.Sync(ctx, request)
	if err != nil || !applied.Changed || confirmations != 1 {
		t.Fatalf("authorized %+v %v confirmations=%d", applied, err, confirmations)
	}
	if member {
		if _, err = os.Stat(filepath.Join(root, "uv.lock")); err != nil {
			t.Fatal("missing shared workspace lock")
		}
		if _, err = os.Stat(filepath.Join(packageRoot, "uv.lock")); !os.IsNotExist(err) {
			t.Fatal("created member-local lock")
		}
		inputs, err := config.ReadPythonInputs(root, "dependency", true)
		if err != nil || inputs.WorkspaceRoot != "." || inputs.WorkspaceSHA256 == "" {
			t.Fatalf("workspace inputs %+v %v", inputs, err)
		}
	}
	if _, err = os.Stat(filepath.Join(packageRoot, "build-executed")); err != nil {
		t.Fatal("authorized backend did not execute")
	}
	var output bytes.Buffer
	code, err := runner.Execute(ctx, runner.Process{Executable: applied.Generation.PythonExecutable, Args: []string{"-I", "-c", "import core_probe; assert core_probe.VALUE == 42"}, Directory: root, Environment: os.Environ(), Stdout: &output, Stderr: &output})
	if err != nil || code != 0 {
		t.Fatalf("built package %d %v %s", code, err, output.String())
	}
	request.ConfirmBuild = func(string) (bool, error) { t.Error("healthy sync asked for build permission"); return false, nil }
	unchanged, err := s.Sync(ctx, request)
	if err != nil || unchanged.Changed {
		t.Fatalf("build no-op %+v %v", unchanged, err)
	}
	// The earlier approval must not authorize a later changed invocation.
	f, err := os.OpenFile(filepath.Join(packageRoot, "pyproject.toml"), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("\n# new invocation\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if local {
		status, err := s.Status(ctx, root)
		if err != nil || status.Environment != "drifted" {
			t.Fatalf("local manifest drift %+v %v", status, err)
		}
		if selected, err := s.SelectRun(ctx, root, false); err == nil {
			selected.Release()
			t.Fatal("run ignored local source metadata change")
		}
		if _, err := s.Sync(ctx, SyncRequest{Directory: root, Locked: true}); err == nil || !strings.HasPrefix(err.Error(), "LOCK_OUT_OF_DATE:") {
			t.Fatalf("locked ignored local metadata: %v", err)
		}
	}
	denied, err = s.Sync(ctx, SyncRequest{Directory: root})
	if !errors.As(err, &needs) || denied.Changed {
		t.Fatalf("approval leaked %+v %v", denied, err)
	}
}
