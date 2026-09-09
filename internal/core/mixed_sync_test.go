package core

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
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

func TestMixedRuntimeSyncRetained(t *testing.T) {
	testMixedRuntimeSync(t, false)
}

func TestMixedProjectSyncRetained(t *testing.T) {
	testMixedRuntimeSync(t, true)
}

func testMixedRuntimeSync(t *testing.T, pythonProject bool) {
	pythonPath, nodePath := os.Getenv("MYENV_TEST_PYTHON_RECORD"), os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if pythonPath == "" || nodePath == "" {
		t.Skip("requires retained Python and Node")
	}
	var python struct{ UV, Python, Version string }
	var node struct {
		Archive  string
		Artifact backend.NodeArtifact
	}
	for path, destination := range map[string]any{pythonPath: &python, nodePath: &node} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(data, destination); err != nil {
			t.Fatal(err)
		}
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	if node.Artifact.Platform != platform {
		t.Fatal("retained Node platform mismatch")
	}
	var wheel bytes.Buffer
	if pythonProject {
		writer := zip.NewWriter(&wheel)
		for name, content := range map[string]string{
			"myenv_probe.py":                       "VALUE = 'installed'\n",
			"myenv_probe-1.0.0.dist-info/METADATA": "Metadata-Version: 2.1\nName: myenv-probe\nVersion: 1.0.0\n",
			"myenv_probe-1.0.0.dist-info/WHEEL":    "Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: py3-none-any\n",
			"myenv_probe-1.0.0.dist-info/RECORD":   "",
		} {
			entry, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pythonProject && r.URL.Path == "/myenv_probe-1.0.0-py3-none-any.whl" {
			w.Write(wheel.Bytes())
			return
		}
		if r.URL.Path != "/node-archive" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, node.Archive)
	}))
	defer server.Close()
	root := t.TempDir()
	declaration := "schema: 1\ntools: {node: '22', python: '3.12'}\n"
	if pythonProject {
		declaration += "python: {project: '.'}\n"
		manifest := fmt.Sprintf("[project]\nname='myenv-mixed-test'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\ndependencies=['myenv-probe @ %s/myenv_probe-1.0.0-py3-none-any.whl']\n[tool.uv]\npackage=false\n", server.URL)
		if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte(declaration), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(filepath.Join(root, "myenv.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := config.Digest(c)
	if err != nil {
		t.Fatal(err)
	}
	a := node.Artifact
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{platform: {Tools: map[string]config.RuntimeLock{
		"node":   {Version: a.Version, NPM: a.NPM, Backend: a.Backend, Evidence: a.Evidence, URL: server.URL + "/node-archive", SHA256: a.SHA256},
		"python": {Version: python.Version, Backend: "uv-" + backend.UVVersion, Evidence: "version"},
	}}}}
	if err = config.WriteNewLock(filepath.Join(root, "myenv.lock"), lock); err != nil {
		t.Fatal(err)
	}
	s := &Service{UV: &backend.UV{Executable: python.UV}, PythonDirectory: filepath.Dir(filepath.Dir(python.Python))}
	ctx := context.Background()
	result, err := s.Sync(ctx, SyncRequest{Directory: root, Locked: !pythonProject, AllowBuild: pythonProject})
	if err != nil || !result.Changed || result.LockChanged != pythonProject || result.NativeLockChanged != pythonProject {
		t.Fatalf("mixed sync %+v %v", result, err)
	}
	selected, err := s.SelectRun(ctx, root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer selected.Release()
	g := selected.Generation
	for _, entry := range []string{g.NodeExecutable, g.PythonExecutable} {
		if !strings.HasPrefix(entry, g.Directory+string(filepath.Separator)) {
			t.Fatal("runtime outside final generation")
		}
	}
	for _, check := range []struct {
		executable, expected string
		args                 []string
	}{
		{g.NodeExecutable, "v" + a.Version, []string{"--version"}},
		{g.PythonExecutable, python.Version, []string{"-I", "-c", "import sys; print('.'.join(map(str,sys.version_info[:3])))"}},
	} {
		var output bytes.Buffer
		code, err := runner.Execute(ctx, runner.Process{Executable: check.executable, Args: check.args, Directory: root, Environment: os.Environ(), Stdout: &output, Stderr: &output})
		if err != nil || code != 0 || strings.TrimSpace(output.String()) != check.expected {
			t.Fatalf("mixed entry %d %v %s", code, err, output.String())
		}
	}
	status, err := s.Status(ctx, root)
	if err != nil || status.Environment != "ready" {
		t.Fatalf("mixed status %+v %v", status, err)
	}
	if pythonProject {
		var output bytes.Buffer
		code, err := runner.Execute(ctx, runner.Process{Executable: g.PythonExecutable, Args: []string{"-I", "-c", "import myenv_probe; print(myenv_probe.VALUE)"}, Directory: root, Environment: os.Environ(), Stdout: &output, Stderr: &output})
		if err != nil || code != 0 || strings.TrimSpace(output.String()) != "installed" {
			t.Fatalf("mixed dependency: %d %v %s", code, err, output.String())
		}
	}
	server.Close()
	s.UV = &backend.UV{Executable: filepath.Join(root, "unavailable-uv")}
	unchanged, err := s.Sync(ctx, SyncRequest{Directory: root, Locked: true})
	if err != nil || unchanged.Changed || unchanged.Generation.ID != g.ID {
		t.Fatalf("mixed no-op %+v %v", unchanged, err)
	}
	measureMixedNoopCLI(t, root, g.ID, pythonProject)
	measureMixedNoopMemory(t, root, g.ID)
	measureMixedStatusCLI(t, root, g.ID)
	measureMixedStatusMemory(t, root, g.ID)
	traceMixedStatus(t, s, root, g.ID)
}
