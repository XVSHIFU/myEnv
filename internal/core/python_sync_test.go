package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
)

func TestSyncRetainedPython(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained managed uv and Python")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ UV, Python, Version string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	manifest := []byte("[project]\nname = 'myenv-core-test'\nversion = '0.1.0'\nrequires-python = '>=3.12,<3.13'\ndependencies = []\n[tool.uv]\npackage = false\n[dependency-groups]\ndev = []\n")
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("myenv.yaml", []byte("schema: 1\ntools: {python: '3.12'}\npython: {project: '.', groups: [dev]}\n"))
	write("pyproject.toml", manifest)
	u := &backend.UV{Executable: record.UV}
	s := &Service{UV: u, PythonDirectory: filepath.Dir(filepath.Dir(record.Python))}
	if archive := os.Getenv("MYENV_TEST_UV_ARCHIVE"); archive != "" {
		s.UV, s.UVArchive = nil, archive
	}
	ctx := context.Background()
	preview, err := s.Sync(ctx, SyncRequest{Directory: root, DryRun: true})
	if err != nil || preview.Plan == nil || preview.Plan.UnresolvedPython != "3.12" || !preview.Plan.NeedsApply {
		t.Fatalf("preview %+v %v", preview, err)
	}
	if _, err = os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatal("dry run created workspace state")
	}
	if _, err = s.Sync(ctx, SyncRequest{Directory: root, Locked: true}); err == nil || !strings.Contains(err.Error(), "LOCK_OUT_OF_DATE") {
		t.Fatalf("missing locks: %v", err)
	}
	var phases []SyncPhase
	progress := func(phase SyncPhase) { phases = append(phases, phase) }
	first, err := s.Sync(ctx, SyncRequest{Directory: root, Progress: progress})
	if err != nil || !first.Changed || !first.LockChanged || !first.NativeLockChanged || first.Generation == nil || first.Generation.PythonExecutable == "" {
		t.Fatalf("first sync %+v: %v", first, err)
	}
	wantPhases := []SyncPhase{PhaseWaiting, PhaseBackend, PhaseResolving, PhasePython, PhaseDependencies, PhaseVerifying, PhasePublishing}
	if !slices.Equal(phases, wantPhases) {
		t.Fatalf("preparation phases: %v", phases)
	}
	if !strings.HasPrefix(first.Generation.PythonExecutable, first.Generation.Directory+string(filepath.Separator)) {
		t.Fatal("venv not in final generation")
	}
	lockBytes, err := os.ReadFile(filepath.Join(root, "myenv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	nativeBytes, err := os.ReadFile(filepath.Join(root, "uv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	// An unusable injected backend proves the healthy no-op never starts uv.
	savedUV := s.UV
	s.UV = &backend.UV{Executable: filepath.Join(root, "absent-uv")}
	phases = nil
	second, err := s.Sync(ctx, SyncRequest{Directory: root, Locked: true, Progress: progress})
	if err != nil || second.Changed || second.LockChanged || second.NativeLockChanged || second.Generation.ID != first.Generation.ID {
		t.Fatalf("no-op %+v: %v", second, err)
	}
	if !slices.Equal(phases, []SyncPhase{PhaseWaiting}) {
		t.Fatalf("no-op reported preparation: %v", phases)
	}
	s.UV = savedUV
	for name, expected := range map[string][]byte{"myenv.lock": lockBytes, "uv.lock": nativeBytes} {
		actual, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("locked modified %s", name)
		}
	}
	status, err := s.Status(ctx, root)
	if err != nil || status.Environment != "ready" {
		t.Fatalf("status %+v %v", status, err)
	}
	selected, err := s.SelectRun(ctx, root, false)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	code, runErr := runner.Execute(ctx, runner.Process{Executable: selected.Generation.PythonExecutable, Args: []string{"-I", "-c", "import sys; print('.'.join(map(str,sys.version_info[:3])))"}, Directory: root, Environment: os.Environ(), Stdout: &output, Stderr: &output})
	if err = selected.Release(); err != nil {
		t.Fatal(err)
	}
	if runErr != nil || code != 0 || strings.TrimSpace(output.String()) != record.Version {
		t.Fatalf("run %d %v %s", code, runErr, output.String())
	}
	write("pyproject.toml", append(append([]byte{}, manifest...), []byte("\n# changed native input\n")...))
	status, err = s.Status(ctx, root)
	if err != nil || status.Environment != "drifted" {
		t.Fatalf("native drift %+v %v", status, err)
	}
	if selected, err = s.SelectRun(ctx, root, false); err == nil {
		selected.Release()
		t.Fatal("run accepted native input drift")
	}
	if _, err = s.Sync(ctx, SyncRequest{Directory: root, Locked: true}); err == nil || !strings.Contains(err.Error(), "LOCK_OUT_OF_DATE") {
		t.Fatalf("locked accepted drift: %v", err)
	}
	updated, err := s.Use(ctx, root, "python@"+record.Version)
	if err != nil || !updated.DeclarationChanged || !updated.Changed || updated.Generation.ID == first.Generation.ID {
		t.Fatalf("updated %+v %v", updated, err)
	}
	write("pyproject.toml", append(append([]byte{}, manifest...), []byte("\n[build-system]\nrequires=[]\nbuild-backend='forbidden_backend'\n")...))
	failed, err := s.Sync(ctx, SyncRequest{Directory: root})
	var needsInput *config.NeedsInput
	if !errors.As(err, &needsInput) || failed.Changed {
		t.Fatalf("build permission %+v %v", failed, err)
	}
	selected, err = s.SelectRun(ctx, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Generation.ID != updated.Generation.ID {
		t.Fatal("failure changed active generation")
	}
	selected.Release()
	rolled, err := s.Rollback(ctx, root)
	if err != nil || rolled.ID != first.Generation.ID {
		t.Fatalf("rollback %+v %v", rolled, err)
	}
	// A native lock can be updated before uv rejects an unavailable group.
	write("pyproject.toml", bytes.Replace(manifest, []byte("0.1.0"), []byte("0.2.0"), 1))
	write("myenv.yaml", []byte("schema: 1\ntools: {python: '3.12'}\npython: {project: './', groups: [missing]}\n"))
	failed, err = s.Sync(ctx, SyncRequest{Directory: root})
	if err == nil || !strings.HasPrefix(err.Error(), "SYNC_FAILED:") || failed.Changed || !failed.LockChanged || !failed.NativeLockChanged {
		t.Fatalf("partial native failure %+v %v", failed, err)
	}
	selected, err = s.SelectRun(ctx, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Generation.ID != first.Generation.ID {
		t.Fatal("native sync failure changed active generation")
	}
	selected.Release()
}
