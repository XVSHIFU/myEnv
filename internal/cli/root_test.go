package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitJSONAndNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".node-version"), []byte("22\n"), 0600); err != nil {
		t.Fatal(err)
	}
	invoke := func() result {
		t.Helper()
		var out, diag bytes.Buffer
		exit := Execute([]string{"-C", dir, "init", "--json", "--no-input"}, bytes.NewReader(nil), &out, &diag, "test")
		if exit != 0 {
			t.Fatalf("exit %d: %s %s", exit, &out, &diag)
		}
		var r result
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if diag.Len() != 0 {
			t.Fatal(diag.String())
		}
		return r
	}
	if r := invoke(); !r.OK || !r.Changed || r.Schema != 1 || r.Error != nil {
		t.Fatalf("bad result: %+v", r)
	}
	path := filepath.Join(dir, "myenv.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before = append(before, []byte("# keep this comment\n")...)
	if err = os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	if r := invoke(); r.Changed {
		t.Fatal("existing config changed")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing bytes overwritten")
	}
}

func TestInitNeedsInput(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		dir := t.TempDir()
		if conflict {
			os.WriteFile(filepath.Join(dir, ".node-version"), []byte("22"), 0600)
			os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("24"), 0600)
		}
		var out, diag bytes.Buffer
		exit := Execute([]string{"init", "-C", dir, "--json", "--no-input"}, bytes.NewReader(nil), &out, &diag, "test")
		var r result
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if exit != 3 || r.OK || r.Error == nil || r.Error.Code != "NEEDS_INPUT" {
			t.Fatalf("exit %d: %s", exit, &out)
		}
		if _, err := os.Stat(filepath.Join(dir, "myenv.yaml")); !os.IsNotExist(err) {
			t.Fatal("created config without required input")
		}
	}
}

func TestPythonProjectDetection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".python-version"), []byte("3.12\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := "[project]\nrequires-python = '>=3.12'\n[dependency-groups]\ndev = []\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	if exit := Execute([]string{"init", "-C", dir, "--json"}, bytes.NewReader(nil), &out, &diag, "test"); exit != 0 {
		t.Fatalf("exit %d: %s %s", exit, &out, &diag)
	}
	b, err := os.ReadFile(filepath.Join(dir, "myenv.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("dev")) || !bytes.Contains(b, []byte(">=3.12")) {
		t.Fatalf("unexpected config: %s", b)
	}
}
