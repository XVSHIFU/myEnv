package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
)

func TestManagedUVMirrorRetained(t *testing.T) {
	archive := os.Getenv("MYENV_TEST_UV_ARCHIVE")
	if archive == "" {
		t.Skip("requires retained uv archive")
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	release, err := backend.FixedUVRelease(platform)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/uv/"+release.Version+"/"+path.Base(release.URL) {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		http.ServeFile(w, r, archive)
	}))
	defer server.Close()
	root := t.TempDir()
	storage := config.UserStorage{Data: filepath.Join(root, "data"), Cache: filepath.Join(root, "cache")}
	service := &Service{Storage: &storage, UVMirror: server.URL + "/uv"}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	installed, err := service.managedUV(ctx, filepath.Join(root, "project-a"), platform)
	if err != nil {
		t.Fatal(err)
	}
	if err := installed.VerifyVersion(ctx); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("downloads: %d", requests.Load())
	}
	server.Close()
	reused, err := service.managedUV(ctx, filepath.Join(root, "project-b"), platform)
	if err != nil || reused.Executable != installed.Executable || requests.Load() != 1 {
		t.Fatalf("offline reuse: %+v %v downloads=%d", reused, err, requests.Load())
	}
	for _, project := range []string{"project-a", "project-b"} {
		if _, err := os.Stat(filepath.Join(root, project)); !os.IsNotExist(err) {
			t.Fatalf("shared install changed project: %v", err)
		}
	}
}
