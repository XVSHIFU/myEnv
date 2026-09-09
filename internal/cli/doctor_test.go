package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"myenv/internal/state"
)

func TestDoctorRunProtectionOutput(t *testing.T) {
	testDoctorRunProtectionOutput(t, false)
}

func TestDoctorGlobalRunProtectionOutput(t *testing.T) {
	testDoctorRunProtectionOutput(t, true)
}

func testDoctorRunProtectionOutput(t *testing.T, global bool) {
	root := t.TempDir()
	userConfig := t.TempDir()
	declaration := filepath.Join(root, "myenv.yaml")
	work := filepath.Join(root, ".myenv")
	hint := "clean --dry-run"
	if global {
		profileRoot := filepath.Join(userConfig, "myenv")
		if err := os.Mkdir(profileRoot, 0700); err != nil {
			t.Fatal(err)
		}
		declaration = filepath.Join(profileRoot, "profile.yaml")
		work = filepath.Join(profileRoot, ".myenv-profile")
		hint = "clean --global --dry-run"
		if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("invalid project ["), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(declaration, []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(work, "state.db")
	ctx := context.Background()
	store, err := state.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// State-only fixture: the absent runtime is not an installation success.
	if err := store.Publish(ctx, state.Generation{ID: "protected", Directory: filepath.Join(work, "protected"), InputDigest: "old", NodeExecutable: filepath.Join(work, "protected", "node")}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireActive(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, structured := range []bool{false, true} {
		args := []string{"-C", root, "doctor"}
		if global {
			args = append(args, "--global")
		}
		if structured {
			args = append(args, "--json")
		}
		var out, diagnostic bytes.Buffer
		var code int
		if artifact := os.Getenv("MYENV_TEST_DOCTOR_EXECUTABLE"); artifact != "" {
			if !filepath.IsAbs(artifact) {
				t.Fatal("doctor artifact must be absolute")
			}
			if global && runtime.GOOS != "windows" {
				t.Skip("artifact global namespace injection currently requires Windows")
			}
			command := exec.CommandContext(ctx, artifact, args...)
			command.Stdout, command.Stderr = &out, &diagnostic
			command.Env = os.Environ()
			if global {
				command.Env = append(command.Env, "APPDATA="+userConfig)
			}
			if err := command.Run(); err != nil {
				t.Fatalf("artifact doctor failed: %v %s", err, &diagnostic)
			}
		} else {
			code = execute(args, bytes.NewReader(nil), &out, &diagnostic, "test", userConfig)
		}
		if code != 0 || diagnostic.Len() != 0 {
			t.Fatalf("doctor: %d %s", code, &diagnostic)
		}
		if structured {
			var response struct {
				OK      bool
				Changed bool
				Data    struct {
					Protection string `json:"run_protection"`
					Detail     string `json:"protection_detail"`
				}
			}
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if !response.OK || response.Changed || response.Data.Protection != "present" || !strings.Contains(response.Data.Detail, hint) {
				t.Fatalf("JSON protection: %+v", response)
			}
		} else if !strings.Contains(out.String(), "Run protection records remain") || !strings.Contains(out.String(), hint) {
			t.Fatalf("text protection: %s", &out)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("doctor changed state: %v", err)
	}
}

func TestDoctorDeepJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	code := Execute([]string{"-C", root, "doctor", "--deep", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var r struct {
		OK      bool `json:"ok"`
		Changed bool `json:"changed"`
		Data    struct {
			CheckLevel string `json:"check_level"`
			Evidence   string `json:"content_evidence"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err, out.String())
	}
	if code != 0 || !r.OK || r.Changed || r.Data.CheckLevel != "deep" || r.Data.Evidence != "missing" || diagnostic.Len() != 0 {
		t.Fatalf("code=%d result=%+v diagnostic=%s", code, r, &diagnostic)
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatal("doctor initialized state", err)
	}
}

func TestRebuildRequiresMatchingLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	code := execute([]string{"-C", root, "sync", "--rebuild", "--locked", "--dry-run", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test", t.TempDir())
	var r result
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err, out.String())
	}
	if code != 1 || r.OK || r.Changed || r.Error == nil || r.Error.Code != "LOCK_OUT_OF_DATE" {
		t.Fatalf("code=%d result=%+v", code, r)
	}
	if _, err := os.Stat(filepath.Join(root, ".myenv")); !os.IsNotExist(err) {
		t.Fatal("dry rebuild initialized state", err)
	}
}
