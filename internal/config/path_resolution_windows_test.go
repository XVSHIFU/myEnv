package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWindowsWorkspacePath(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "MixedCase", "nested", "project")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "pyproject.toml")
	if err := os.WriteFile(file, []byte("[project]\nname='probe'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	compare := func(t *testing.T, path string) {
		t.Helper()
		want, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := resolveWorkspacePath(path)
		if err != nil || got != want {
			t.Fatalf("%q: handle=%q, EvalSymlinks=%q, error=%v", path, got, want, err)
		}
	}
	for _, path := range []string{root, dir, file, strings.ToLower(file)} {
		compare(t, path)
	}
	if _, err := resolveWorkspacePath(filepath.Join(dir, "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing path error: %v", err)
	}
	t.Run("symlink", func(t *testing.T) {
		link := filepath.Join(root, "link")
		if err := os.Symlink(dir, link); err != nil {
			t.Skipf("symlink creation unavailable: %v", err)
		}
		compare(t, link)
		compare(t, filepath.Join(link, "pyproject.toml"))
	})
	t.Run("junction", func(t *testing.T) {
		outside := t.TempDir()
		link := filepath.Join(root, "junction")
		command := exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference='Stop'; New-Item -ItemType Junction -Path $env:MYENV_JUNCTION_LINK -Target $env:MYENV_JUNCTION_TARGET | Out-Null")
		command.Env = append(os.Environ(), "MYENV_JUNCTION_LINK="+link, "MYENV_JUNCTION_TARGET="+outside)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("create isolated junction: %v: %s", err, output)
		}
		// Remove only the junction entry before TempDir cleanup of its target.
		t.Cleanup(func() {
			if err := os.Remove(link); err != nil {
				t.Error(err)
			}
		})
		resolved, err := resolveWorkspacePath(link)
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil || (relative != ".." && !strings.HasPrefix(relative, `..\`)) {
			t.Fatalf("junction escape not visible: %q %v", relative, err)
		}
		for _, relative := range []string{"junction", "junction/missing/child"} {
			got, err := Within(root, relative)
			if err == nil {
				t.Fatalf("accepted junction escape %q: %q", relative, got)
			}
		}
		want, err := resolveWorkspacePath(outside)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := Within(link, "."); err != nil || got != want {
			t.Fatalf("junction root: %q %v", got, err)
		}
		// Exercise callers as well as the path helper: an escaped local source
		// or glob member must fail instead of being omitted from input digests.
		if err := os.WriteFile(filepath.Join(outside, "pyproject.toml"), []byte("[project]\nname='outside'\nversion='1'\n"), 0600); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ name, project, manifest string }{
			{"project", "junction", "[project]\nname='root'\nversion='1'\n"},
			{"source", ".", "[project]\nname='root'\nversion='1'\n[tool.uv.sources]\nprobe={path='junction'}\n"},
			{"member", ".", "[tool.uv.workspace]\nmembers=['junction']\n"},
			{"recursive_member", ".", "[tool.uv.workspace]\nmembers=['junction/**']\n"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(tc.manifest), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := ReadPythonInputs(root, tc.project, false); err == nil || !strings.Contains(err.Error(), "escapes workspace") {
					t.Fatalf("expected boundary rejection, got %v", err)
				}
			})
		}
		// A dangling junction still denotes a link, not a safe missing child.
		if err := os.Remove(filepath.Join(outside, "pyproject.toml")); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(outside); err != nil {
			t.Fatal(err)
		}
		for _, relative := range []string{"junction", "junction/missing/child"} {
			if got, err := Within(root, relative); err == nil {
				t.Fatalf("accepted dangling junction %q: %q", relative, got)
			}
		}
	})
	t.Run("long_path", func(t *testing.T) {
		long := dir
		for len(long) < 550 {
			long = filepath.Join(long, strings.Repeat("x", 50))
		}
		if err := os.MkdirAll(long, 0700); err != nil {
			t.Fatal(err)
		}
		compare(t, long)
	})
	// Paired local observations are diagnostic, not full-CLI performance gates.
	if os.Getenv("MYENV_TEST_HANDLE_PATH") != "1" {
		return
	}
	var handleTime, evalTime time.Duration
	for i := 0; i < 101; i++ {
		measure := func(handle bool) {
			start := time.Now()
			var err error
			if handle {
				_, err = resolveWorkspacePath(file)
			} else {
				_, err = filepath.EvalSymlinks(file)
			}
			elapsed := time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			if i > 0 {
				if handle {
					handleTime += elapsed
				} else {
					evalTime += elapsed
				}
			}
		}
		measure(i%2 == 0)
		measure(i%2 != 0)
	}
	t.Logf("100 paired calls, local existing file: handle mean=%s, EvalSymlinks mean=%s", handleTime/100, evalTime/100)
}
