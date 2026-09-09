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
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestSyncRetainedNode(t *testing.T) {
	testSyncRetainedNode(t, false)
}

func TestNodeRebuildRetained(t *testing.T) { testSyncRetainedNode(t, true) }

func testSyncRetainedNode(t *testing.T, rebuildOnly bool) {
	recordPath := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if recordPath == "" {
		t.Skip("requires retained verified Node archive")
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Artifact backend.NodeArtifact
		Archive  string
	}
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	if record.Artifact.Platform != platform {
		t.Fatalf("retained artifact %s does not match host %s", record.Artifact.Platform, platform)
	}
	fileKey := map[string]string{"windows-amd64": "win-x64-zip", "linux-amd64-glibc": "linux-x64", "darwin-arm64": "osx-arm64-tar"}[platform]
	filename := path.Base(record.Artifact.URL)
	var requests atomic.Int64
	var downloads atomic.Int64
	var fail atomic.Bool
	var changeLock atomic.Bool
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if fail.Load() {
			http.Error(w, "injected download failure", 503)
			return
		}
		switch {
		case r.URL.Path == "/index.json":
			fmt.Fprintf(w, `[{"version":"v%s","npm":"%s","files":["%s"]}]`, record.Artifact.Version, record.Artifact.NPM, fileKey)
		case strings.HasSuffix(r.URL.Path, "SHASUMS256.txt"):
			fmt.Fprintf(w, "%s  %s\n", record.Artifact.SHA256, filename)
		case path.Base(r.URL.Path) == filename:
			downloads.Add(1)
			if changeLock.Load() {
				if err := os.WriteFile(filepath.Join(root, "myenv.lock"), []byte("{\"schema\":0}\n"), 0600); err != nil {
					t.Error(err)
				}
			}
			http.ServeFile(w, r, record.Archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	configPath := filepath.Join(root, "myenv.yaml")
	write := func(extra string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte("schema: 1\ntools: {node: \"22\"}\n"+extra), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("")
	service := Service{Node: &backend.Node{Client: server.Client(), BaseURL: server.URL}}
	if rebuildOnly {
		storage := t.TempDir()
		service.Storage = &config.UserStorage{Data: filepath.Join(storage, "data"), Cache: filepath.Join(storage, "cache")}
	}
	ctx := context.Background()
	preview, err := service.Sync(ctx, SyncRequest{Directory: root, DryRun: true})
	if err != nil || preview.Changed || preview.Plan == nil || !preview.Plan.NeedsApply || preview.Plan.Node.Version != record.Artifact.Version {
		t.Fatalf("initial preview: %+v %v", preview, err)
	}
	if downloads.Load() != 0 {
		t.Fatal("preview downloaded runtime")
	}
	for _, name := range []string{".myenv", "myenv.lock"} {
		if _, err = os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("preview created %s: %v", name, err)
		}
	}
	first, err := service.Sync(ctx, SyncRequest{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Generation == nil {
		t.Fatal("first sync did not publish")
	}
	if rebuildOnly {
		started := time.Now()
		diagnosis, err := service.DoctorDeep(ctx, root, os.Environ())
		t.Logf("first Node deep check elapsed: %s", time.Since(started))
		if err != nil || diagnosis.ContentEvidence != "matched" {
			t.Fatalf("Node baseline: %+v %v", diagnosis, err)
		}
		npmCLI := filepath.Join(filepath.Dir(first.Generation.NodeExecutable), "node_modules", "npm", "bin", "npm-cli.js")
		if platform != "windows-amd64" {
			npmCLI = filepath.Join(filepath.Dir(filepath.Dir(first.Generation.NodeExecutable)), "lib", "node_modules", "npm", "bin", "npm-cli.js")
		}
		original, err := os.ReadFile(npmCLI)
		if err != nil {
			t.Fatal(err)
		}
		changed := append(append([]byte(nil), original...), []byte("\n// myenv drift fixture\n")...)
		if err = os.WriteFile(npmCLI, changed, 0600); err != nil {
			t.Fatal(err)
		}
		diagnosis, err = service.DoctorDeep(ctx, root, os.Environ())
		if err != nil || diagnosis.ContentEvidence != "mismatch" {
			t.Fatalf("npm drift: %+v %v", diagnosis, err)
		}
		lockBefore, err := os.ReadFile(filepath.Join(root, "myenv.lock"))
		if err != nil {
			t.Fatal(err)
		}
		server.Close() // Rebuild must reuse the verified shared archive.
		rebuilt, err := service.Sync(ctx, SyncRequest{Directory: root, Locked: true, Rebuild: true})
		if err != nil || !rebuilt.Changed || rebuilt.LockChanged || rebuilt.Generation.ID == first.Generation.ID {
			t.Fatalf("Node rebuild: %+v %v", rebuilt, err)
		}
		lockAfter, err := os.ReadFile(filepath.Join(root, "myenv.lock"))
		if err != nil || !bytes.Equal(lockBefore, lockAfter) {
			t.Fatal("rebuild changed Node lock", err)
		}
		if current, err := os.ReadFile(npmCLI); err != nil || !bytes.Equal(current, changed) {
			t.Fatal("rebuild changed old npm", err)
		}
		diagnosis, err = service.DoctorDeep(ctx, root, os.Environ())
		if err != nil || diagnosis.ContentEvidence != "matched" {
			t.Fatalf("rebuilt Node evidence: %+v %v", diagnosis, err)
		}
		if downloads.Load() != 1 {
			t.Fatal("rebuild redownloaded cached archive")
		}
		return
	}
	before := requests.Load()
	stateFiles := []string{filepath.Join(root, "myenv.yaml"), filepath.Join(root, "myenv.lock"), filepath.Join(root, ".myenv", "state.db")}
	saved := make([][]byte, len(stateFiles))
	for i, file := range stateFiles {
		saved[i], err = os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
	}
	preview, err = service.Sync(ctx, SyncRequest{Directory: root, DryRun: true, Locked: true})
	if err != nil || preview.Changed || preview.Plan == nil || preview.Plan.NeedsApply || requests.Load() != before {
		t.Fatalf("applied preview: %+v %v", preview, err)
	}
	for i, file := range stateFiles {
		data, err := os.ReadFile(file)
		if err != nil || !bytes.Equal(data, saved[i]) {
			t.Fatalf("preview changed %s: %v", file, err)
		}
	}
	second, err := service.Sync(ctx, SyncRequest{Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Generation.ID != first.Generation.ID || requests.Load() != before {
		t.Fatal("unchanged sync performed work")
	}
	unchangedUse, err := service.Use(ctx, root, "node@22")
	if err != nil || unchangedUse.DeclarationChanged || unchangedUse.Changed || unchangedUse.LockChanged || requests.Load() != before {
		t.Fatalf("unchanged use performed work: %+v %v", unchangedUse, err)
	}
	used, err := service.Use(ctx, root, "node@"+record.Artifact.Version)
	if err != nil || !used.DeclarationChanged || !used.Changed || !used.LockChanged || used.Generation == nil || used.Generation.ID == first.Generation.ID {
		t.Fatalf("use did not apply exact declaration: %+v %v", used, err)
	}
	selectedUse, err := service.SelectRun(ctx, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if selectedUse.Generation.ID != used.Generation.ID || selectedUse.Config.Tools["node"] != record.Artifact.Version {
		t.Fatal("use did not publish requested declaration snapshot")
	}
	if err = selectedUse.Release(); err != nil {
		t.Fatal(err)
	}
	first = used.SyncResult
	if err = os.Remove(filepath.Join(first.Generation.Directory, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	repaired, err := service.Sync(ctx, SyncRequest{Directory: root, Locked: true})
	if err != nil {
		t.Fatal(err)
	}
	if !repaired.Changed || repaired.Generation.ID == first.Generation.ID {
		t.Fatal("missing snapshot did not cause rebuild")
	}
	selectedRepaired, err := service.SelectRun(ctx, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if selectedRepaired.Generation.ID != repaired.Generation.ID {
		t.Fatal("repair did not publish selected generation")
	}
	if err = selectedRepaired.Release(); err != nil {
		t.Fatal(err)
	}
	first = repaired
	write("env: {APP_ENV: changed}\n")
	if _, err = service.SelectRun(ctx, root, false); err == nil {
		t.Fatal("run accepted declaration drift")
	}
	selected, err := service.SelectRun(ctx, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Config.Env["APP_ENV"] != "" {
		t.Fatal("current selected new configuration instead of applied snapshot")
	}
	if err = selected.Release(); err != nil {
		t.Fatal(err)
	}
	fail.Store(true)
	if _, err = service.Sync(ctx, SyncRequest{Directory: root}); err == nil {
		t.Fatal("injected failure succeeded")
	}
	store, err := state.Open(ctx, filepath.Join(root, ".myenv", "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	active, err := store.Active(ctx)
	if err != nil || active == nil || active.ID != first.Generation.ID {
		t.Fatalf("old generation lost: %+v %v", active, err)
	}
	output, err := exec.Command(active.NodeExecutable, "--version").Output()
	if err != nil || strings.TrimSpace(string(output)) != "v"+record.Artifact.Version {
		t.Fatalf("old runtime unusable: %s %v", output, err)
	}
	fail.Store(false)
	changeLock.Store(true)
	if _, err = service.Sync(ctx, SyncRequest{Directory: root, Locked: true}); err == nil || !strings.HasPrefix(err.Error(), "INPUT_CHANGED:") {
		t.Fatalf("lock edit during preparation: %v", err)
	}
	active, err = store.Active(ctx)
	if err != nil || active == nil || active.ID != first.Generation.ID {
		t.Fatalf("lock race replaced active generation: %+v %v", active, err)
	}
	changed, err := os.ReadFile(filepath.Join(root, "myenv.lock"))
	if err != nil || string(changed) != "{\"schema\":0}\n" {
		t.Fatalf("overwrote external lock edit: %q %v", changed, err)
	}
}
