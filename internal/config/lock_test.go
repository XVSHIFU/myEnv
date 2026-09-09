package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLockEvidenceAndNoOverwrite(t *testing.T) {
	lock := &Lock{Schema: 1, ConfigDigest: strings.Repeat("a", 64), Platforms: map[string]PlatformLock{"windows-amd64": {Tools: map[string]RuntimeLock{"node": {Version: "22.1.0", Backend: "node-official-v1", Evidence: "artifact", URL: "https://nodejs.org/dist/example.zip", SHA256: strings.Repeat("b", 64)}}}}}
	path := filepath.Join(t.TempDir(), "myenv.lock")
	if err := WriteNewLock(path, lock); err != nil {
		t.Fatal(err)
	}
	if err := WriteNewLock(path, lock); err == nil {
		t.Fatal("overwrote existing lock")
	}
	loaded, err := ReadLock(path)
	if err != nil {
		t.Fatal(err)
	}
	node := loaded.Platforms["windows-amd64"].Tools["node"]
	node.SHA256 = ""
	loaded.Platforms["windows-amd64"].Tools["node"] = node
	if err = loaded.Validate(); err == nil {
		t.Fatal("accepted artifact evidence without checksum")
	}
	if err = ReplaceLock(path, loaded); err == nil {
		t.Fatal("replaced with invalid lock")
	}
	retained, err := ReadLock(path)
	if err != nil || retained.ConfigDigest != lock.ConfigDigest {
		t.Fatal("invalid replacement damaged old lock")
	}
	lock.ConfigDigest = strings.Repeat("c", 64)
	if err = ReplaceLock(path, lock); err != nil {
		t.Fatal(err)
	}
	replaced, err := ReadLock(path)
	if err != nil || replaced.ConfigDigest != lock.ConfigDigest {
		t.Fatalf("replacement not visible: %v", err)
	}
}

func TestConfigDigestStableMapOrder(t *testing.T) {
	a := &Config{Schema: 1, Tools: map[string]string{"node": "22", "python": "3.12"}}
	b := &Config{Schema: 1, Tools: map[string]string{"python": "3.12", "node": "22"}}
	aa, _ := Digest(a)
	bb, _ := Digest(b)
	if aa != bb {
		t.Fatal("map order changed digest")
	}
	b.Tools["node"] = "24"
	bb, _ = Digest(b)
	if aa == bb {
		t.Fatal("version edit missed")
	}
}

func TestPythonLockInputs(t *testing.T) {
	inputs := &PythonInputs{Project: ".", PyprojectSHA256: strings.Repeat("a", 64), UVLockSHA256: strings.Repeat("b", 64)}
	lock := &Lock{Schema: 1, ConfigDigest: strings.Repeat("c", 64), Platforms: map[string]PlatformLock{"windows-amd64": {Tools: map[string]RuntimeLock{"python": {Version: "3.12.13", Backend: "uv-test", Evidence: "version"}}, Python: inputs}}}
	path := filepath.Join(t.TempDir(), "myenv.lock")
	if err := WriteNewLock(path, lock); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadLock(path)
	if err != nil || *loaded.Platforms["windows-amd64"].Python != *inputs {
		t.Fatalf("native lock roundtrip %+v %v", loaded, err)
	}
	for _, field := range []*string{&inputs.PyprojectSHA256, &inputs.UVLockSHA256} {
		prior := *field
		*field = ""
		if err := lock.Validate(); err == nil {
			t.Fatal("accepted missing required native digest")
		}
		*field = prior
	}
	inputs.Project = "../outside"
	if err := lock.Validate(); err == nil {
		t.Fatal("accepted escaping native project")
	}
}
