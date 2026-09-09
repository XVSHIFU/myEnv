package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myenv/internal/core"
)

func TestDoctorCanceledJSON(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, diagnostic bytes.Buffer
	code := ExecuteContext(ctx, []string{"-C", filepath.Join(t.TempDir(), "missing"), "doctor", "--deep", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var r result
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err, out.String())
	}
	if code != 130 || r.OK || r.Changed || r.Error == nil || r.Error.Code != "CANCELED" || diagnostic.Len() != 0 {
		t.Fatalf("code=%d result=%+v diagnostic=%s", code, r, &diagnostic)
	}
}

type cancelOnCleanItem struct {
	bytes.Buffer
	cancel context.CancelFunc
}

func (w *cancelOnCleanItem) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if bytes.Contains(p, []byte(`"node_archive"`)) {
		w.cancel()
	}
	return n, err
}

func TestCleanCanceledPartialJSON(t *testing.T) {
	namespace := t.TempDir()
	s, err := runtimeService(false, namespace)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(s.Storage.Cache, "node-archives")
	if err = os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	for _, letter := range []string{"a", "b"} {
		if err = os.WriteFile(filepath.Join(cache, strings.Repeat(letter, 64)), []byte("cache"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &cancelOnCleanItem{cancel: cancel}
	var diagnostic bytes.Buffer
	code := executeContext(ctx, []string{"clean", "--cache", "node", "--json"}, bytes.NewReader(nil), out, &diagnostic, "test", namespace)
	var r struct {
		OK      bool     `json:"ok"`
		Changed bool     `json:"changed"`
		Error   *failure `json:"error"`
		Data    struct {
			Items   []core.CleanItem `json:"items"`
			Summary core.CleanResult `json:"summary"`
		} `json:"data"`
	}
	if err = json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err, out.String())
	}
	if code != 130 || r.OK || !r.Changed || r.Error == nil || r.Error.Code != "CANCELED" || len(r.Data.Items) != 1 || r.Data.Summary.Removed != 1 || diagnostic.Len() != 0 {
		t.Fatalf("code=%d result=%+v diagnostic=%s", code, r, &diagnostic)
	}
	remaining, err := s.CleanNodeCache(context.Background(), true, nil)
	if err != nil || remaining.Candidates != 1 {
		t.Fatalf("remaining=%+v %v", remaining, err)
	}
}

func TestInitAlreadyCanceledJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".node-version"), []byte("22\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, diagnostic bytes.Buffer
	code := ExecuteContext(ctx, []string{"-C", root, "init", "--json"}, bytes.NewReader(nil), &out, &diagnostic, "test")
	var response result
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if code != 130 || response.OK || response.Changed || response.Error == nil || response.Error.Code != "CANCELED" {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != ".node-version" {
		t.Fatalf("canceled init created files: %v %v", entries, err)
	}
}
