package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"myenv/internal/backend"
)

func TestGlobalProfileCLIRetained(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained Python and uv")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ UV, Python, Version string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	userConfig, project := t.TempDir(), t.TempDir()
	// --global must not parse this unrelated project declaration.
	if err = os.WriteFile(filepath.Join(project, "myenv.yaml"), []byte("not valid YAML ["), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := runtimeService(true, userConfig)
	if err != nil {
		t.Fatal(err)
	}
	s.UV = &backend.UV{Executable: record.UV}
	s.PythonDirectory = filepath.Dir(filepath.Dir(record.Python))
	first, err := s.Use(context.Background(), project, "python@3.12")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Use(context.Background(), project, "python@"+record.Version); err != nil {
		t.Fatal(err)
	}
	call := func(args ...string) (int, []byte, string) {
		t.Helper()
		var out, diagnostic bytes.Buffer
		code := execute(append([]string{"-C", project}, args...), bytes.NewReader(nil), &out, &diagnostic, "test", userConfig)
		return code, out.Bytes(), diagnostic.String()
	}
	code, out, diagnostic := call("run", "--global", "python", "-X", "utf8", "-c", "import sys; print(sys.argv[1],end=''); sys.exit(17)", "--global")
	if code != 17 || string(out) != "--global" || diagnostic != "" {
		t.Fatalf("global run %d %s %s", code, out, diagnostic)
	}
	for _, args := range [][]string{{"sync", "--global", "--locked", "--json"}, {"use", "--global", "python@" + record.Version, "--json"}, {"doctor", "--global", "--json"}} {
		code, out, diagnostic := call(args...)
		var response result
		if err = json.Unmarshal(out, &response); err != nil {
			t.Fatal(err)
		}
		if code != 0 || !response.OK || response.Changed {
			t.Fatalf("global command %v: %d %s %s", args, code, out, diagnostic)
		}
	}
	code, out, diagnostic = call("rollback", "--global", "--json")
	if code != 0 {
		t.Fatalf("global rollback %d %s %s", code, out, diagnostic)
	}
	selected, err := s.SelectRun(context.Background(), project, true)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Generation.ID != first.Generation.ID {
		t.Fatal("CLI rollback selected wrong profile generation")
	}
	selected.Release()
	if code, _, _ := call("run", "--global", "python", "--version"); code != 1 {
		t.Fatalf("profile drift exit=%d", code)
	}
	if code, out, diagnostic := call("run", "--global", "--current", "python", "--version"); code != 0 {
		t.Fatalf("profile current %d %s %s", code, out, diagnostic)
	}
	if code, _, _ := call("init", "--global", "--json"); code != 2 {
		t.Fatal("init accepted unsupported global scope")
	}
	if _, err = os.Stat(filepath.Join(project, ".myenv")); !os.IsNotExist(err) {
		t.Fatal("global CLI initialized project state")
	}
}
