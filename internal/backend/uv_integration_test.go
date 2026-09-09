package backend

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myenv/internal/runner"
)

func TestUVOfficialDownload(t *testing.T) {
	if os.Getenv("MYENV_TEST_UV_DOWNLOAD") != "1" {
		t.Skip("explicit official uv download gate")
	}
	root := os.Getenv("MYENV_TEST_ARTIFACTS")
	if !filepath.IsAbs(root) {
		t.Fatal("MYENV_TEST_ARTIFACTS must be an absolute isolated directory")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "uv-download.json")); err == nil {
		t.Fatal("retained download already exists; reuse its record")
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	release, archive, err := DownloadUV(ctx, platform, root)
	if err != nil {
		t.Fatal(err)
	}
	record := struct {
		Release UVRelease
		Archive string
	}{release, archive}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "uv-download.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("retained SHA256-verified uv %s archive at %s", release.Version, archive)
}

func TestUVRetainedPrepare(t *testing.T) {
	path := os.Getenv("MYENV_TEST_UV_RECORD")
	if path == "" {
		t.Skip("explicit retained uv preparation gate")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Release UVRelease
		Archive string
	}
	if err = json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	if record.Release.Platform != platform {
		t.Fatal("record platform differs from host")
	}
	directory := filepath.Join(filepath.Dir(path), "prepared-"+time.Now().Format("20060102T150405.000000000"))
	u, err := PrepareUV(context.Background(), record.Archive, directory, platform)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.MarshalIndent(u, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(filepath.Dir(path), "uv-prepared.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("verified managed uv executable %s", u.Executable)
}
