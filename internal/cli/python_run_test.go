package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/core"
	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestRunRetainedPython(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained managed Python venv")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ Executable, Version string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	work := filepath.Join(root, ".myenv")
	if err = os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {python: \"3.12\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &config.Config{Schema: 1, Tools: map[string]string{"python": "3.12"}}
	digest, _ := config.Digest(c)
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	lock := &config.Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]config.PlatformLock{platform: {Tools: map[string]config.RuntimeLock{"python": {Version: record.Version, Backend: "uv-" + backend.UVVersion, Evidence: "version"}}}}}
	for _, directory := range []string{root, work} {
		if err = config.WriteNewLock(filepath.Join(directory, "myenv.lock"), lock); err != nil {
			t.Fatal(err)
		}
	}
	if err = config.WriteSnapshot(work, c); err != nil {
		t.Fatal(err)
	}
	if err = config.MarkComplete(work, digest); err != nil {
		t.Fatal(err)
	}
	s, err := state.Open(context.Background(), filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Publish(context.Background(), state.Generation{ID: "python", Directory: work, InputDigest: digest, PythonExecutable: record.Executable}, ""); err != nil {
		t.Fatal(err)
	}
	s.Close()
	var out, diagnostic bytes.Buffer
	code := Execute([]string{"-C", root, "run", "python", "-X", "utf8", "-c", "import sys; print(sys.argv[1],end=''); sys.exit(17)", "参数 with spaces"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if code != 17 || out.String() != "参数 with spaces" || diagnostic.Len() != 0 {
		t.Fatalf("Python exit=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	out.Reset()
	diagnostic.Reset()
	code = Execute([]string{"-C", root, "doctor", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var diagnosis struct {
		OK, Changed bool
		Data        core.Diagnosis
	}
	if err = json.Unmarshal(out.Bytes(), &diagnosis); err != nil {
		t.Fatal(err)
	}
	if code != 0 || !diagnosis.OK || diagnosis.Changed || diagnosis.Data.Environment != "ready" || diagnosis.Data.AppliedPython != record.Executable || diagnosis.Data.AppliedNode != "" {
		t.Fatalf("Python doctor exit=%d stdout=%s stderr=%s", code, out.String(), diagnostic.String())
	}
	actual, actualErr := os.Stat(diagnosis.Data.RunPathPython)
	want, wantErr := os.Stat(record.Executable)
	if actualErr != nil || wantErr != nil || !os.SameFile(actual, want) {
		t.Fatal("doctor resolved a different Python executable")
	}
}
