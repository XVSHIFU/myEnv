package cli

import (
	"bytes"
	"encoding/json"
	"myenv/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemRemoveLastProfileJSON(t *testing.T) {
	root := t.TempDir()
	path, err := config.ProfilePath(root)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, []byte("schema: 1\ntools: {go: '1.26.6'}\n"), 0600)
	var out, diagnostic bytes.Buffer
	code := execute([]string{"system", "remove", "go", "--json"}, strings.NewReader(""), &out, &diagnostic, "test", root)
	if code != 0 {
		t.Fatalf("%d: %s %s", code, &out, &diagnostic)
	}
	var response result
	if err = json.Unmarshal(out.Bytes(), &response); err != nil || !response.OK || !response.Changed {
		t.Fatalf("invalid response %s: %v", &out, err)
	}
	c, err := config.LoadProfile(path)
	if err != nil || len(c.Tools) != 0 {
		t.Fatalf("profile %+v %v", c, err)
	}
}
func TestSystemHelpUsesSystemScope(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := execute([]string{"--lang", "zh-CN", "system", "doctor", "--help"}, strings.NewReader(""), &out, &diagnostic, "test", t.TempDir())
	if code != 0 || !strings.Contains(out.String(), "发现已安装环境") || strings.Contains(out.String(), "记录的内容基线") {
		t.Fatalf("wrong nested help: %s %s", &out, &diagnostic)
	}
}

func TestSystemHelpIncludesTranslatedHelpCommand(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := execute([]string{"--lang", "zh-CN", "--help"}, strings.NewReader(""), &out, &diagnostic, "test", t.TempDir())
	if code != 0 || strings.Contains(out.String(), "Read command help") || !strings.Contains(out.String(), "阅读命令帮助或离线手册") {
		t.Fatalf("%s %s", &out, &diagnostic)
	}
}
