package core

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
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

// This registry serves actual installable wheels with deterministic test
// modules. No build hook or external registry is needed to verify group rules.
func TestPythonWheelGroupsRetained(t *testing.T) {
	testPythonWheelGroups(t, false, false)
}

func TestPythonWheelLocalConfigRetained(t *testing.T) {
	testPythonWheelGroups(t, true, false)
}

func TestPythonWheelCacheCleanupRetained(t *testing.T) { testPythonWheelGroups(t, false, true) }

func TestPythonWheelRebuildRetained(t *testing.T) { testPythonWheelGroups(t, true, true) }

func testPythonWheelGroups(t *testing.T, localConfig, cleanCache bool) {
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
	wheels := map[string][]byte{}
	for _, name := range []string{"myenv_base", "myenv_qa", "myenv_dev"} {
		var archive bytes.Buffer
		writer := zip.NewWriter(&archive)
		for file, content := range map[string]string{
			name + ".py":                       "VALUE = '" + name + "'\n",
			name + "-1.0.0.dist-info/METADATA": "Metadata-Version: 2.1\nName: " + strings.ReplaceAll(name, "_", "-") + "\nVersion: 1.0.0\n",
			name + "-1.0.0.dist-info/WHEEL":    "Wheel-Version: 1.0\nGenerator: myenv-test\nRoot-Is-Purelib: true\nTag: py3-none-any\n",
			name + "-1.0.0.dist-info/RECORD":   "",
		} {
			entry, err := writer.Create(file)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = entry.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
		}
		if err = writer.Close(); err != nil {
			t.Fatal(err)
		}
		wheels[name] = archive.Bytes()
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, wheel := range wheels {
			filename := name + "-1.0.0-py3-none-any.whl"
			if r.URL.Path == "/simple/"+strings.ReplaceAll(name, "_", "-")+"/" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<a href="/files/%s#sha256=%x">%s</a>`, filename, sha256.Sum256(wheel), filename)
				return
			}
			if r.URL.Path == "/files/"+filename {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(wheel)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	if err = os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(parent, "uv.toml"), []byte("invalid parent configuration ["), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("[project]\nname='myenv-group-project'\nversion='0.1.0'\nrequires-python='>=3.12,<3.13'\ndependencies=['myenv-base==1.0.0']\n[dependency-groups]\ndev=['myenv-dev==1.0.0']\nqa=['myenv-qa==1.0.0']\n[tool.uv]\npackage=false\ndefault-groups=['dev']\n[[tool.uv.index]]\nurl='%s/simple'\ndefault=true\n", server.URL)
	if err = os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if localConfig {
		if err = os.Mkdir(filepath.Join(root, "wheels"), 0700); err != nil {
			t.Fatal(err)
		}
		for name, wheel := range wheels {
			if err = os.WriteFile(filepath.Join(root, "wheels", name+"-1.0.0-py3-none-any.whl"), wheel, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err = os.WriteFile(filepath.Join(root, "uv.toml"), []byte("no-index=true\nfind-links=['./wheels']\n"), 0600); err != nil {
			t.Fatal(err)
		}
		server.Close() // uv.toml must override the project HTTP index.
	}
	declare := func(groups string) {
		t.Helper()
		declaration := "schema: 1\ntools: {python: '3.12'}\npython: {project: '.', groups: [" + groups + "]}\n"
		if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte(declaration), 0600); err != nil {
			t.Fatal(err)
		}
	}
	declare("qa")
	s := &Service{UV: &backend.UV{Executable: record.UV}, PythonDirectory: filepath.Dir(filepath.Dir(record.Python))}
	if cleanCache {
		s.Storage = &config.UserStorage{Data: filepath.Join(parent, "data"), Cache: filepath.Join(parent, "cache")}
	}
	ctx := context.Background()
	first, err := s.Sync(ctx, SyncRequest{Directory: root})
	if err != nil || !first.Changed {
		t.Fatalf("wheel sync %+v %v", first, err)
	}
	verify := func(executable string, qa bool) {
		t.Helper()
		script := "import myenv_base,importlib.util; assert myenv_base.VALUE=='myenv_base'; assert importlib.util.find_spec('myenv_dev') is None; "
		if qa {
			script += "import myenv_qa; assert myenv_qa.VALUE=='myenv_qa'"
		} else {
			script += "assert importlib.util.find_spec('myenv_qa') is None"
		}
		var output bytes.Buffer
		code, err := runner.Execute(ctx, runner.Process{Executable: executable, Args: []string{"-I", "-c", script}, Directory: root, Environment: os.Environ(), Stdout: &output, Stderr: &output})
		if err != nil || code != 0 {
			t.Fatalf("wheel imports %d %v %s", code, err, output.String())
		}
	}
	verify(first.Generation.PythonExecutable, true)
	nativeBefore, err := os.ReadFile(filepath.Join(root, "uv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	// The native lock resolves all groups; environment selection remains explicit.
	for name, wheel := range wheels {
		evidence := fmt.Sprintf("%x", sha256.Sum256(wheel))
		if localConfig {
			evidence = `path = "` + name + `-1.0.0-py3-none-any.whl"`
		}
		if !bytes.Contains(nativeBefore, []byte(strings.ReplaceAll(name, "_", "-"))) || !bytes.Contains(nativeBefore, []byte(evidence)) {
			t.Fatalf("native lock lacks wheel evidence for %s: %s", name, nativeBefore)
		}
	}
	if localConfig && !bytes.Contains(nativeBefore, []byte(`registry = "wheels"`)) {
		t.Fatal("relative wheel directory resolved against the wrong configuration origin")
	}
	leftovers, err := filepath.Glob(filepath.Join(root, ".myenv-uv-config-*.toml"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("configuration snapshot left behind: %v %v", leftovers, err)
	}
	declare("")
	if selected, err := s.SelectRun(ctx, root, false); err == nil {
		selected.Release()
		t.Fatal("run accepted changed groups")
	}
	second, err := s.Sync(ctx, SyncRequest{Directory: root})
	if err != nil || !second.Changed || second.Generation.ID == first.Generation.ID {
		t.Fatalf("group change %+v %v", second, err)
	}
	verify(second.Generation.PythonExecutable, false)
	verify(first.Generation.PythonExecutable, true)
	nativeAfter, err := os.ReadFile(filepath.Join(root, "uv.lock"))
	if err != nil || !bytes.Equal(nativeBefore, nativeAfter) {
		t.Fatal("group selection unnecessarily changed native lock")
	}
	server.Close()
	if cleanCache {
		preview, err := s.CleanUVCache(ctx, true, nil)
		if err != nil || preview.Bytes == 0 || preview.Changed {
			t.Fatalf("populated cache preview %+v %v", preview, err)
		}
		cleaned, err := s.CleanUVCache(ctx, false, nil)
		if err != nil || !cleaned.Changed {
			t.Fatalf("cache cleanup %+v %v", cleaned, err)
		}
		// Both current and rollback generations must import actual installed
		// wheel content with the registry shut down and the cache removed.
		verify(first.Generation.PythonExecutable, true)
		verify(second.Generation.PythonExecutable, false)
		selected, err := s.SelectRun(ctx, root, false)
		if err != nil {
			t.Fatal("cache cleanup invalidated active environment", err)
		}
		if selected.Generation.ID != second.Generation.ID {
			selected.Release()
			t.Fatal("cache cleanup changed active generation")
		}
		selected.Release()
	}
	unchanged, err := s.Sync(ctx, SyncRequest{Directory: root, Locked: true})
	if err != nil || unchanged.Changed {
		t.Fatalf("offline no-op %+v %v", unchanged, err)
	}
	if localConfig && cleanCache {
		// Real installed dependency bytes, rather than an unrelated added file,
		// must be diagnosed and replaced by a locked rebuild from local wheels.
		var installed bytes.Buffer
		code, err := runner.Execute(ctx, runner.Process{Executable: second.Generation.PythonExecutable, Args: []string{"-I", "-c", "import myenv_base; print(myenv_base.__file__,end='')"}, Directory: root, Environment: os.Environ(), Stdout: &installed})
		if err != nil || code != 0 {
			t.Fatalf("locate module: %d %v", code, err)
		}
		relative, err := filepath.Rel(second.Generation.Directory, installed.String())
		if err != nil {
			t.Fatal(err)
		}
		module, err := config.Within(second.Generation.Directory, relative)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(module, []byte("VALUE = 'changed'\n"), 0600); err != nil {
			t.Fatal(err)
		}
		diagnosis, err := s.DoctorDeep(ctx, root, os.Environ())
		if err != nil || diagnosis.ContentEvidence != "mismatch" {
			t.Fatalf("dependency drift: %+v %v", diagnosis, err)
		}
		lockBefore, err := os.ReadFile(filepath.Join(root, "myenv.lock"))
		if err != nil {
			t.Fatal(err)
		}
		rebuilt, err := s.Sync(ctx, SyncRequest{Directory: root, Locked: true, Rebuild: true})
		if err != nil || !rebuilt.Changed || rebuilt.Generation.ID == second.Generation.ID || rebuilt.LockChanged || rebuilt.NativeLockChanged {
			t.Fatalf("dependency rebuild: %+v %v", rebuilt, err)
		}
		verify(rebuilt.Generation.PythonExecutable, false)
		verify(first.Generation.PythonExecutable, true)
		if contents, err := os.ReadFile(module); err != nil || string(contents) != "VALUE = 'changed'\n" {
			t.Fatal("rebuild modified old dependency", err)
		}
		lockAfter, err := os.ReadFile(filepath.Join(root, "myenv.lock"))
		if err != nil || !bytes.Equal(lockBefore, lockAfter) {
			t.Fatal("rebuild changed runtime lock", err)
		}
		nativeAfter, err := os.ReadFile(filepath.Join(root, "uv.lock"))
		if err != nil || !bytes.Equal(nativeBefore, nativeAfter) {
			t.Fatal("rebuild changed native lock", err)
		}
		diagnosis, err = s.DoctorDeep(ctx, root, os.Environ())
		if err != nil || diagnosis.ContentEvidence != "matched" {
			t.Fatalf("rebuilt dependency evidence: %+v %v", diagnosis, err)
		}
	}
}
