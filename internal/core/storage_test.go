package core

import (
	"context"
	"encoding/json"
	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSharedUVRetained(t *testing.T) {
	archive := os.Getenv("MYENV_TEST_UV_ARCHIVE")
	if archive == "" {
		t.Skip("requires retained uv archive")
	}
	root := t.TempDir()
	storage, err := config.ResolveUserStorage(filepath.Join(root, "data"), filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	workA, workB := filepath.Join(root, "project-a"), filepath.Join(root, "project-b")
	type outcome struct {
		uv  backend.UV
		err error
	}
	results := make(chan outcome, 2)
	for _, work := range []string{workA, workB} {
		go func(work string) {
			u, err := (&Service{Storage: &storage, UVArchive: archive}).managedUV(context.Background(), work, platform)
			results <- outcome{u, err}
		}(work)
	}
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.uv.Executable != b.uv.Executable {
		t.Fatalf("shared preparation %+v %+v", a, b)
	}
	relative, err := filepath.Rel(storage.Data, a.uv.Executable)
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := config.Within(storage.Data, relative); err != nil || resolved != a.uv.Executable {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(storage.Data, "backends"))
	if err != nil {
		t.Fatal(err)
	}
	prepared := 0
	for _, e := range entries {
		if e.IsDir() {
			prepared++
		}
	}
	if prepared != 1 {
		t.Fatalf("prepared %d backend copies", prepared)
	}
	reused, err := (&Service{Storage: &storage, UVArchive: filepath.Join(root, "missing-archive")}).managedUV(context.Background(), workB, platform)
	if err != nil || reused.Executable != a.uv.Executable {
		t.Fatalf("reuse %+v %v", reused, err)
	}
	for _, work := range []string{workA, workB} {
		if _, err = os.Stat(work); !os.IsNotExist(err) {
			t.Fatal("shared backend wrote project directory", err)
		}
	}
}

func TestSharedPythonConcurrentRetained(t *testing.T) {
	path := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if path == "" {
		t.Skip("requires retained uv/Python")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct{ UV, Python, Version string }
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	storage := config.UserStorage{Data: filepath.Join(root, "data"), Cache: filepath.Join(root, "cache")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	type outcome struct {
		python, executable string
		err                error
	}
	results := make(chan outcome, 2)
	for _, name := range []string{"a", "b"} {
		go func(name string) {
			s := &Service{Storage: &storage, PythonDirectory: filepath.Dir(filepath.Dir(record.Python))}
			python, executable, err := s.preparePython(ctx, backend.UV{Executable: record.UV}, record.Version, filepath.Join(root, name), filepath.Join(root, name, "generation"), platform)
			results <- outcome{python, executable, err}
		}(name)
	}
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.python != b.python || a.executable == b.executable {
		t.Fatalf("concurrent Python %+v %+v", a, b)
	}
	for _, entry := range []string{a.executable, b.executable} {
		if info, err := os.Stat(entry); err != nil || !info.Mode().IsRegular() {
			t.Fatal("missing independent venv", err)
		}
	}
}
