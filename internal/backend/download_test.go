package backend

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDownloadChecksumAndCleanup(t *testing.T) {
	payload := "test archive bytes"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, payload) }))
	defer server.Close()
	n := &Node{Client: server.Client()}
	dir := t.TempDir()
	artifact := NodeArtifact{URL: server.URL, SHA256: strings.Repeat("0", 64)}
	if _, err := n.Download(context.Background(), artifact, dir); err == nil || !strings.Contains(err.Error(), "CHECKSUM_MISMATCH") {
		t.Fatalf("mismatch: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("left partial archive: %v %v", entries, err)
	}
	artifact.SHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	path, err := n.Download(context.Background(), artifact, dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != payload {
		t.Fatalf("download content: %q %v", got, err)
	}
}
