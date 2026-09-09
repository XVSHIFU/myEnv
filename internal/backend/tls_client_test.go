package backend

import (
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomCABundleInputBounds(t *testing.T) {
	root := t.TempDir()
	empty := filepath.Join(root, "empty.pem")
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}
	large := filepath.Join(root, "large.pem")
	file, err := os.Create(large)
	if err != nil {
		t.Fatal(err)
	}
	truncateErr := file.Truncate((4 << 20) + 1)
	closeErr := file.Close()
	if truncateErr != nil || closeErr != nil {
		t.Fatal(truncateErr, closeErr)
	}
	for _, check := range []struct{ name, path, reason string }{
		{"empty", empty, "no valid PEM"},
		{"directory", root, "regular PEM file"},
		{"oversized", large, "4 MiB"},
		{"missing", filepath.Join(root, "missing.pem"), ""},
	} {
		t.Run(check.name, func(t *testing.T) {
			client, err := DownloadClient(check.path)
			if client != nil || err == nil || !strings.Contains(err.Error(), check.reason) {
				t.Fatalf("invalid bundle: client=%v error=%v", client, err)
			}
			if check.name == "missing" && !os.IsNotExist(err) {
				t.Fatalf("missing-file cause lost: %v", err)
			}
		})
	}
}

type opaqueCATransport struct{}

func (opaqueCATransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("unexpected network call")
}

func TestCustomCARejectsOpaqueTransport(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(file, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	for _, transport := range []http.RoundTripper{opaqueCATransport{}, (*http.Transport)(nil), nil} {
		http.DefaultTransport = transport
		if client, err := DownloadClient(file); err == nil || client != nil {
			t.Fatalf("unsupported transport accepted: %T %v", transport, err)
		}
	}
}

func TestDownloadClientExplicitCA(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(file, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := DownloadClient(file)
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatal(response.Status)
	}
	plain, err := DownloadClient("")
	if err != nil {
		t.Fatal(err)
	}
	if response, err := plain.Get(server.URL); err == nil {
		response.Body.Close()
		t.Fatal("custom trust leaked into default client")
	}
	if err := os.WriteFile(file, []byte("not a certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := DownloadClient(file); err == nil {
		t.Fatal("invalid certificate accepted")
	}
	if _, err := DownloadClient("relative.pem"); err == nil {
		t.Fatal("relative CA path accepted")
	}
}
