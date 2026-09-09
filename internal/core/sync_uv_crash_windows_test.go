package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestSyncKilledDuringRealUVRequest(t *testing.T) {
	testSyncKilledDuringRealUVRequest(t, false)
}

func TestSyncKilledRealUVPreservesActive(t *testing.T) {
	testSyncKilledDuringRealUVRequest(t, true)
}

func testSyncKilledDuringRealUVRequest(t *testing.T, existing bool) {
	record := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if record == "" {
		t.Skip("requires retained Windows uv and Python")
	}
	var python struct{ UV, Python, Version string }
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &python); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if root := os.Getenv("MYENV_TEST_REAL_UV_OWNER"); root != "" {
		service := &Service{UV: &backend.UV{Executable: python.UV}, PythonDirectory: filepath.Dir(filepath.Dir(python.Python))}
		_, err := service.Sync(ctx, SyncRequest{Directory: root, AllowBuild: true})
		t.Fatalf("Sync unexpectedly returned before kill: %v", err)
	}
	requested, disconnected, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var requestOnce, disconnectOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/myenv_crash_probe-1.0.0-py3-none-any.whl" {
			http.NotFound(w, r)
			return
		}
		requestOnce.Do(func() { close(requested) })
		select {
		case <-r.Context().Done():
			disconnectOnce.Do(func() { close(disconnected) })
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	root := t.TempDir()
	for name, contents := range map[string]string{
		"myenv.yaml":     "schema: 1\ntools: {python: '3.12'}\npython: {project: '.'}\n",
		"pyproject.toml": fmt.Sprintf("[project]\nname='myenv-crash-test'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\ndependencies=['myenv-crash-probe @ %s/myenv_crash_probe-1.0.0-py3-none-any.whl']\n[tool.uv]\npackage=false\n", server.URL),
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := config.Load(filepath.Join(root, "myenv.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := config.Digest(c)
	if err != nil {
		t.Fatal(err)
	}
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{"windows-amd64": {Tools: map[string]config.RuntimeLock{
		"python": {Version: python.Version, Backend: "uv-" + backend.UVVersion, Evidence: "version"},
	}}}}
	if err := config.WriteNewLock(filepath.Join(root, "myenv.lock"), lock); err != nil {
		t.Fatal(err)
	}
	var activeID string
	if existing {
		manifestPath := filepath.Join(root, "pyproject.toml")
		manifest, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		dependency := fmt.Sprintf("'myenv-crash-probe @ %s/myenv_crash_probe-1.0.0-py3-none-any.whl'", server.URL)
		if err := os.WriteFile(manifestPath, []byte(strings.Replace(string(manifest), dependency, "", 1)), 0600); err != nil {
			t.Fatal(err)
		}
		service := &Service{UV: &backend.UV{Executable: python.UV}, PythonDirectory: filepath.Dir(filepath.Dir(python.Python))}
		baseline, err := service.Sync(ctx, SyncRequest{Directory: root, AllowBuild: true})
		if err != nil || baseline.Generation == nil || !baseline.Changed {
			t.Fatalf("baseline: %+v %v", baseline, err)
		}
		activeID = baseline.Generation.ID
		if err := os.WriteFile(manifestPath, manifest, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+t.Name()+"$")
	command.Env = append(os.Environ(), "MYENV_TEST_REAL_UV_OWNER="+root)
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	waited := false
	defer func() {
		if !waited {
			_ = command.Process.Kill()
			<-done
		}
	}()
	select {
	case <-requested:
	case err := <-done:
		waited = true
		t.Fatalf("Sync exited before uv request: %v\n%s", err, output.String())
	case <-ctx.Done():
		t.Fatal("real uv never requested local dependency")
	}
	store, err := state.OpenReadOnly(ctx, filepath.Join(root, ".myenv", "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	owners, err := store.PreparingChildOperations(ctx, "")
	if err != nil || len(owners) != 1 || !owners[0].Tracked {
		store.Close()
		t.Fatalf("tracked Sync owner: %+v %v", owners, err)
	}
	children, err := store.PreparationChildren(ctx, owners[0].ID, "")
	store.Close()
	pending := 0
	for _, child := range children {
		if !child.Completed {
			pending++
		}
	}
	if err != nil || pending != 1 {
		t.Fatalf("running uv registration: %+v %v", children, err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil {
		t.Fatal("Sync owner was not killed")
	}
	waited = true
	select {
	case <-disconnected:
	case <-ctx.Done():
		t.Fatal("uv connection survived owner death")
	}
	preview, err := (&Service{}).Clean(ctx, root, true, nil)
	if err != nil || preview.Candidates != 1 || preview.Changed {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	actual, err := (&Service{}).Clean(ctx, root, false, nil)
	if err != nil || actual.RecoveredPreparations != 1 || actual.Removed != 1 || actual.Bytes != preview.Bytes {
		t.Fatalf("recovery: %+v %v", actual, err)
	}
	if existing {
		selected, err := (&Service{}).SelectRun(ctx, root, true)
		if err != nil {
			t.Fatal(err)
		}
		if selected.Generation.ID != activeID {
			selected.Release()
			t.Fatal("crashed sync replaced the active generation")
		}
		var version bytes.Buffer
		code, runErr := runner.Execute(ctx, runner.Process{Executable: selected.Generation.PythonExecutable, Args: []string{"-I", "-c", "import sys; print('.'.join(map(str,sys.version_info[:3])))"}, Environment: os.Environ(), Stdout: &version, Stderr: &version, TreeID: selected.TreeID, Completion: selected.Completion})
		releaseErr := selected.Finish(runErr)
		if runErr != nil || releaseErr != nil || code != 0 || strings.TrimSpace(version.String()) != python.Version {
			t.Fatalf("old active Python: %d %v %v %s", code, runErr, releaseErr, version.String())
		}
	}
}
