package core

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
)

func TestProfilePythonRetained(t *testing.T) {
	recordPath := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if recordPath == "" {
		t.Skip("requires retained Python and uv")
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ UV, Python, Version string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	userConfig := t.TempDir()
	profilePath, err := config.ProfilePath(userConfig)
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Dir(profilePath)
	if err = os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	declaration := []byte("schema: 1\ntools: {node: '24'}\nenv: {PROJECT_ONLY: 'yes'}\n")
	projectPath := filepath.Join(project, "myenv.yaml")
	if err = os.WriteFile(projectPath, declaration, 0600); err != nil {
		t.Fatal(err)
	}
	u := &backend.UV{Executable: record.UV}
	storage, err := config.ResolveUserStorage(filepath.Join(userConfig, "data"), filepath.Join(userConfig, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{Storage: &storage, Profile: true, UserConfigDirectory: userConfig, UV: u, PythonDirectory: filepath.Dir(filepath.Dir(record.Python))}
	ctx := context.Background()
	first, err := s.Use(ctx, project, "python@3.12")
	if err != nil || !first.DeclarationChanged || !first.Changed {
		t.Fatalf("profile creation %+v %v", first, err)
	}
	selected, err := s.SelectRun(ctx, project, false)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Config.Tools["node"] != "" || len(selected.Config.Env) != 0 || selected.Generation.PythonExecutable == "" {
		t.Fatalf("profile inherited project: %+v", selected.Config)
	}
	selected.Release()
	diagnosis, err := s.DoctorDeep(ctx, project, os.Environ())
	if err != nil || diagnosis.ContentEvidence != "matched" || diagnosis.CheckLevel != "deep" {
		t.Fatalf("published profile evidence: %+v %v", diagnosis, err)
	}
	unexpected := filepath.Join(first.Generation.Directory, "unexpected.txt")
	if err = os.WriteFile(unexpected, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	diagnosis, err = s.DoctorDeep(ctx, project, os.Environ())
	if err != nil || diagnosis.ContentEvidence != "mismatch" || diagnosis.Environment != "content_changed" {
		t.Fatalf("changed profile evidence: %+v %v", diagnosis, err)
	}
	quick, err := s.Doctor(ctx, project, os.Environ())
	if err != nil || quick.CheckLevel != "quick" || quick.ContentEvidence != "" || quick.Environment != "ready" {
		t.Fatalf("quick check scanned generation: %+v %v", quick, err)
	}
	// Explicit rebuild repairs owned content without changing the lock or
	// mutating the old generation, including when preparation fails first.
	lockPath := s.desiredLockPath(project)
	lockBefore, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	s.UV = &backend.UV{Executable: filepath.Join(project, "absent-uv")}
	preview, err := s.Sync(ctx, SyncRequest{Directory: project, Locked: true, Rebuild: true, DryRun: true})
	if err != nil || preview.Plan == nil || !preview.Plan.NeedsApply || preview.Changed {
		t.Fatalf("rebuild preview: %+v %v", preview, err)
	}
	failedRebuild, err := s.Sync(ctx, SyncRequest{Directory: project, Locked: true, Rebuild: true})
	if err == nil || failedRebuild.Changed {
		t.Fatalf("failed rebuild: %+v %v", failedRebuild, err)
	}
	still, err := s.Status(ctx, project)
	if err != nil || still.Generation.ID != first.Generation.ID {
		t.Fatalf("rebuild failure replaced active: %+v %v", still, err)
	}
	s.UV = u
	rebuilt, err := s.Sync(ctx, SyncRequest{Directory: project, Locked: true, Rebuild: true})
	if err != nil || !rebuilt.Changed || rebuilt.Generation.ID == first.Generation.ID || rebuilt.LockChanged || rebuilt.NativeLockChanged {
		t.Fatalf("rebuild: %+v %v", rebuilt, err)
	}
	lockAfter, err := os.ReadFile(lockPath)
	if err != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("locked rebuild changed lock", err)
	}
	if data, err := os.ReadFile(unexpected); err != nil || string(data) != "changed" {
		t.Fatal("rebuild mutated old generation", err)
	}
	if _, err := os.Stat(filepath.Join(rebuilt.Generation.Directory, "unexpected.txt")); !os.IsNotExist(err) {
		t.Fatal("rebuild copied drift", err)
	}
	diagnosis, err = s.DoctorDeep(ctx, project, os.Environ())
	if err != nil || diagnosis.ContentEvidence != "matched" {
		t.Fatalf("rebuilt evidence: %+v %v", diagnosis, err)
	}
	first.Generation = rebuilt.Generation
	if selected, err := (&Service{}).SelectRun(ctx, project, false); err == nil {
		selected.Release()
		t.Fatal("project inherited profile state")
	}
	s.UV = &backend.UV{Executable: filepath.Join(project, "absent-uv")}
	noop, err := s.Use(ctx, project, "python@3.12")
	if err != nil || noop.DeclarationChanged || noop.Changed {
		t.Fatalf("profile no-op %+v %v", noop, err)
	}
	s.UV = u
	second, err := s.Use(ctx, project, "python@"+record.Version)
	if err != nil || !second.Changed || second.Generation.ID == first.Generation.ID {
		t.Fatalf("profile switch %+v %v", second, err)
	}
	before, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	rolled, err := s.Rollback(ctx, project)
	if err != nil || rolled.ID != first.Generation.ID {
		t.Fatalf("profile rollback %+v %v", rolled, err)
	}
	after, err := os.ReadFile(profilePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rollback changed profile declaration")
	}
	if selected, err := s.SelectRun(ctx, project, false); err == nil {
		selected.Release()
		t.Fatal("ordinary profile run accepted rollback drift")
	}
	selected, err = s.SelectRun(ctx, project, true)
	if err != nil {
		t.Fatal(err)
	}
	selected.Release()
	actual, err := os.ReadFile(projectPath)
	if err != nil || !bytes.Equal(actual, declaration) {
		t.Fatal("profile modified adjacent project")
	}
	for _, name := range []string{"myenv.lock", ".myenv"} {
		if _, err := os.Stat(filepath.Join(project, name)); !os.IsNotExist(err) {
			t.Fatalf("profile created project state %s", name)
		}
	}
	// A failed edit must retain the actual usable generation, not merely its row.
	s.UV = &backend.UV{Executable: filepath.Join(project, "absent-uv")}
	failed, err := s.Use(ctx, project, "python@3.12.0")
	if err == nil || !failed.DeclarationChanged || failed.Changed || !strings.Contains(err.Error(), "myenv sync --global") || !strings.Contains(err.Error(), "myenv run --global --current") {
		t.Fatalf("profile failure result %+v %v", failed, err)
	}
	edited, err := config.LoadProfile(profilePath)
	if err != nil || edited.Tools["python"] != "3.12.0" {
		t.Fatalf("failed use lost requested declaration %+v %v", edited, err)
	}
	status, err := s.Status(ctx, project)
	if err != nil || status.Scope != "profile" || status.Environment != "drifted" || !strings.Contains(status.NextAction, "myenv sync --global") {
		t.Fatalf("profile failure status %+v %v", status, err)
	}
	selected, err = s.SelectRun(ctx, project, true)
	if err != nil {
		t.Fatal(err)
	}
	defer selected.Release()
	if selected.Generation.ID != first.Generation.ID {
		t.Fatal("failed use replaced active generation")
	}
	output, err := exec.CommandContext(ctx, selected.Generation.PythonExecutable, "-I", "-c", "print(42,end='')").CombinedOutput()
	if err != nil || string(output) != "42" {
		t.Fatalf("retained profile no longer runs: %s %v", output, err)
	}
}
