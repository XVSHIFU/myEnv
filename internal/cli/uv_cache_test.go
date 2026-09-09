package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"myenv/internal/backend"
	"myenv/internal/runner"
	"os"
	"path/filepath"
	"testing"
)

func TestUVCleanCLIRetained(t *testing.T) {
	recordPath := os.Getenv("MYENV_TEST_UV_PREPARED")
	if recordPath == "" {
		t.Skip("requires retained uv")
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ Executable string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	namespace := t.TempDir()
	s, err := runtimeService(false, namespace)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(s.Storage.Cache, "uv")
	fixture := filepath.Join(cache, "archive-v0", "fixture")
	if err = os.MkdirAll(fixture, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(fixture, "data")
	if err = os.WriteFile(file, []byte("cache"), 0600); err != nil {
		t.Fatal(err)
	}
	call := func(dry bool, want int, changed bool) {
		t.Helper()
		args := []string{"clean", "--cache", "uv", "--json"}
		if dry {
			args = append(args, "--dry-run")
		}
		var out, diag bytes.Buffer
		code := execute(args, bytes.NewReader(nil), &out, &diag, "test", namespace)
		var r result
		if err = json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err, out.String())
		}
		if code != want || r.Changed != changed || r.OK != (want == 0) {
			t.Fatalf("result %d %+v %s", code, r, diag.String())
		}
	}
	call(true, 0, false)
	call(false, 1, false)
	if _, err = os.Stat(s.Storage.Data); !os.IsNotExist(err) {
		t.Fatal("cleanup initialized backend storage", err)
	}
	if _, err = os.Stat(file); err != nil {
		t.Fatal("preview or failed clean removed cache", err)
	}
	backends := filepath.Join(s.Storage.Data, "backends")
	if err = os.MkdirAll(backends, 0700); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(record.Executable)
	input, err := os.Open(record.Executable)
	if err != nil {
		t.Fatal(err)
	}
	output, err := os.OpenFile(filepath.Join(backends, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		input.Close()
		t.Fatal(err)
	}
	_, err = io.Copy(output, input)
	input.Close()
	closeErr := output.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(backends, "uv-"+backend.UVVersion+"-"+platform), []byte(name+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	call(false, 0, true)
	if _, err = os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("uv did not clean fixture", err)
	}
	if _, err = os.Stat(filepath.Join(backends, name)); err != nil {
		t.Fatal("cleanup removed backend", err)
	}
}
