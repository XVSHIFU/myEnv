package config

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// This measures bounded lock parsing, not process startup or command p95.
func BenchmarkReadLock(b *testing.B) {
	digest := strings.Repeat("a", 64)
	lock := &Lock{Schema: 1, ConfigDigest: digest, Platforms: map[string]PlatformLock{}}
	for _, platform := range []string{"windows-amd64", "linux-amd64", "darwin-arm64"} {
		lock.Platforms[platform] = PlatformLock{Tools: map[string]RuntimeLock{
			"node":   {Version: "22.23.2", Backend: "node-official-v1", Evidence: "artifact", URL: "https://example.test/node.zip", SHA256: digest, NPM: "10.9.8"},
			"python": {Version: "3.12.13", Backend: "uv-0.11.26", Evidence: "version"},
		}, Python: &PythonInputs{ConfigPolicy: "project-only-v1", Project: ".", PyprojectSHA256: digest, UVLockSHA256: digest}}
	}
	path := filepath.Join(b.TempDir(), "myenv.lock")
	if err := WriteNewLock(path, lock); err != nil {
		b.Fatal(err)
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		b.Fatal(err)
	}
	b.Run("strict_keys", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := checkJSONKeys(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("file_decode_validate", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := ReadLock(path); err != nil {
				b.Fatal(err)
			}
		}
	})
}
