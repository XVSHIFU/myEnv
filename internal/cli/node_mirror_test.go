package cli

import (
	"bytes"
	"context"
	"crypto/tls"
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
)

func TestNodeMirrorRealSync(t *testing.T) {
	testNodeMirrorRealSync(t, false)
}

func TestNodeMirrorCustomCARealSync(t *testing.T) {
	testNodeMirrorRealSync(t, true)
}

func testNodeMirrorRealSync(t *testing.T, secure bool) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Windows Node archive")
	}
	var prepared struct {
		Archive  string
		Artifact backend.NodeArtifact
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	a := prepared.Artifact
	filename := "node-v" + a.Version + "-win-x64.zip"
	fileKind := "win-x64-zip"
	switch a.Platform {
	case "windows-amd64":
	case "linux-amd64-glibc":
		filename = "node-v" + a.Version + "-linux-x64.tar.gz"
		fileKind = "linux-x64"
	default:
		t.Fatalf("unsupported retained platform %s", a.Platform)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dist/index.json":
			fmt.Fprintf(w, `[{"version":"v%s","npm":"%s","files":["%s"]}]`, a.Version, a.NPM, fileKind)
		case "/dist/v" + a.Version + "/SHASUMS256.txt":
			fmt.Fprintf(w, "%s  %s\n", a.SHA256, filename)
		case "/dist/v" + a.Version + "/" + filename:
			http.ServeFile(w, r, prepared.Archive)
		default:
			http.NotFound(w, r)
		}
	}))
	if secure {
		certificate, rootPEM := mirrorCertificate(t)
		server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
		server.StartTLS()
		certificateFile := filepath.Join(t.TempDir(), "ca.pem")
		if err := os.WriteFile(certificateFile, rootPEM, 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SSL_CERT_FILE", certificateFile)
	} else {
		server.Start()
	}
	defer server.Close()
	t.Setenv("MYENV_NODE_MIRROR", server.URL+"/dist")
	root, namespace := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	if code := execute([]string{"-C", root, "sync", "--no-input"}, bytes.NewReader(nil), &out, &diagnostic, "test", namespace); code != 0 {
		t.Fatalf("sync: %d %s %s", code, out.String(), diagnostic.String())
	}
	lock, err := config.ReadLock(filepath.Join(root, "myenv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	locked := lock.Platforms[a.Platform].Tools["node"]
	if locked.URL != server.URL+"/dist/v"+a.Version+"/"+filename || locked.SHA256 != a.SHA256 {
		t.Fatalf("mirror lock: %+v", locked)
	}
	server.Close()
	out.Reset()
	diagnostic.Reset()
	if code := execute([]string{"-C", root, "run", "node", "--version"}, bytes.NewReader(nil), &out, &diagnostic, "test", namespace); code != 0 || strings.TrimSpace(out.String()) != "v"+a.Version {
		t.Fatalf("run: %d %s %s", code, out.String(), diagnostic.String())
	}
}

func TestExplicitNodeMirror(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dist/index.json":
			fmt.Fprint(w, `[{"version":"v22.9.0","npm":"10.9.0","files":["win-x64-zip"]}]`)
		case "/dist/v22.9.0/SHASUMS256.txt":
			fmt.Fprintf(w, "%s  node-v22.9.0-win-x64.zip\n", strings.Repeat("a", 64))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("MYENV_NODE_MIRROR", server.URL+"/dist/")
	service, err := runtimeService(false, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := service.Node.Resolve(context.Background(), "22", "windows-amd64")
	if err != nil || artifact.URL != server.URL+"/dist/v22.9.0/node-v22.9.0-win-x64.zip" || artifact.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("mirror resolution: %+v %v", artifact, err)
	}
}

func TestNodeMirrorRejectsInvalidAndSecretValues(t *testing.T) {
	for _, value := range []string{"relative", "file:///tmp/node", "https://user:secret@example.test", "https://example.test?token=secret", "https://example.test/#secret", "https://example.test?"} {
		t.Setenv("MYENV_NODE_MIRROR", value)
		if _, err := runtimeService(false, t.TempDir()); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("invalid mirror leaked or accepted: %v", err)
		}
	}
	t.Setenv("MYENV_NODE_MIRROR", "")
	service, err := runtimeService(false, t.TempDir())
	if err != nil || service.Node != nil {
		t.Fatalf("default backend changed: %+v %v", service, err)
	}
}

func TestExplicitUVMirror(t *testing.T) {
	t.Setenv("MYENV_PYTHON_MIRROR", "https://mirror.example/python/")
	t.Setenv("MYENV_UV_MIRROR", "https://mirror.example/uv/")
	service, err := runtimeService(false, t.TempDir())
	if err != nil || service.UVMirror != "https://mirror.example/uv" || service.PythonMirror != "https://mirror.example/python" {
		t.Fatalf("uv mirror: %+v %v", service, err)
	}
	t.Setenv("MYENV_UV_MIRROR", "https://user:secret@mirror.example")
	if _, err := runtimeService(false, t.TempDir()); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("uv mirror credential rejection: %v", err)
	}
}
