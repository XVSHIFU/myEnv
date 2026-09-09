package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPythonMirrorRealInstall(t *testing.T) {
	record, archive := os.Getenv("MYENV_TEST_PYTHON_RECORD"), os.Getenv("MYENV_TEST_PYTHON_ARCHIVE")
	if record == "" || archive == "" {
		t.Skip("requires retained uv metadata and Python archive")
	}
	var prepared struct{ UV, Version string }
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/python/") || !strings.Contains(r.URL.Path, "cpython-"+prepared.Version) {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		http.ServeFile(w, r, archive)
	}))
	defer server.Close()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	u := UV{Executable: prepared.UV}
	python, err := u.InstallPythonFromMirror(ctx, prepared.Version, filepath.Join(root, "runtimes"), filepath.Join(root, "cache"), server.URL+"/python")
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("runtime downloads: %d", requests.Load())
	}
	server.Close()
	reused, err := u.InstallPythonFromMirror(ctx, prepared.Version, filepath.Join(root, "runtimes"), filepath.Join(root, "cache"), server.URL+"/python")
	if err != nil || reused != python {
		t.Fatalf("offline reuse: %q %v", reused, err)
	}
	if err := u.CreateVenv(ctx, python, filepath.Join(root, "venv"), filepath.Join(root, "cache")); err != nil {
		t.Fatal(err)
	}
}

func TestPythonMirrorRealUVRequest(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PYTHON_RECORD")
	if record == "" {
		t.Skip("requires retained uv/Python metadata")
	}
	var prepared struct{ UV, Version string }
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	requested := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requested <- r.URL.Path:
		default:
		}
		cancel()
		<-r.Context().Done()
	}))
	defer server.Close()
	root := t.TempDir()
	_, err = (UV{Executable: prepared.UV}).InstallPythonFromMirror(ctx, prepared.Version, filepath.Join(root, "runtimes"), filepath.Join(root, "cache"), server.URL+"/python")
	if err == nil {
		t.Fatal("canceled mirror download succeeded")
	}
	select {
	case request := <-requested:
		if !strings.HasPrefix(request, "/python/") || !strings.Contains(request, "cpython-"+prepared.Version) {
			t.Fatalf("unexpected runtime URL: %q", request)
		}
	default:
		t.Fatalf("uv did not use mirror: %v", err)
	}
}
