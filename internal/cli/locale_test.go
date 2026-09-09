package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestLocaleSelectionAndRunBoundary(t *testing.T) {
	t.Setenv("MYENV_LANG", "zh-CN")
	for _, tc := range []struct {
		args    []string
		chinese bool
		invalid bool
	}{
		{nil, true, false}, {[]string{"--lang=en", "help"}, false, false},
		{[]string{"--lang", "zh-CN", "run", "node", "--lang=en"}, true, false},
		{[]string{"-C", "run", "--lang=en", "help"}, false, false},
		{[]string{"run", "-C", "中文目录", "--lang=en", "node", "--lang=zh-CN"}, false, false},
		{[]string{"run", "--", "node", "--lang=en"}, true, false},
		{[]string{"--lang=fr", "help"}, false, true},
	} {
		got, err := selectLocale(tc.args)
		if bool(got) != tc.chinese || (err != nil) != tc.invalid {
			t.Fatalf("%v: %v %v", tc.args, got, err)
		}
	}
}

func TestLocalizedHelpAndJSONContract(t *testing.T) {
	t.Setenv("MYENV_LANG", "en")
	invoke := func(args ...string) (int, string, string) {
		var out, diag bytes.Buffer
		code := Execute(args, strings.NewReader(""), &out, &diag, "test")
		if !utf8.Valid(out.Bytes()) || !utf8.Valid(diag.Bytes()) {
			t.Fatal("invalid UTF-8")
		}
		return code, out.String(), diag.String()
	}
	for _, args := range [][]string{{"--lang=zh-CN", "--help"}, {"help", "manual", "--lang=zh-CN"}, {"run", "--lang=zh-CN", "--help"}} {
		code, out, diag := invoke(args...)
		if code != 0 || !strings.Contains(out, "用法：") || strings.Contains(out, "Available Commands:") || diag != "" {
			t.Fatalf("%v: %d %s %s", args, code, out, diag)
		}
	}
	_, manual, _ := invoke("--lang=zh-CN", "help", "manual")
	for _, topic := range []string{"监督与删除保护", "SSL_CERT_FILE", "--lang", "命令参考", "卸载"} {
		if !strings.Contains(manual, topic) {
			t.Fatal(topic)
		}
	}
	_, en, _ := invoke("--lang=en", "--help")
	if !strings.Contains(en, "Available Commands:") {
		t.Fatal(en)
	}
	for _, args := range [][]string{{"--bad-option", "--json"}, {"run", "--lang=zh-CN", "--json", "node"}} {
		_, a, _ := invoke(append([]string{"--lang=en"}, args...)...)
		_, b, _ := invoke(append([]string{"--lang=zh-CN"}, args...)...)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(a), &parsed); err != nil || a != b {
			t.Fatalf("JSON drift: %s / %s (%v)", a, b, err)
		}
	}
	_, _, diag := invoke("--lang=zh-CN", "unknown-command")
	if !strings.Contains(diag, "USAGE_ERROR：") || !strings.Contains(diag, "原始原因：") || !strings.Contains(diag, "myenv --help") {
		t.Fatal(diag)
	}
	// Generation must remain byte-for-byte stable across UI languages.
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		_, a, _ := invoke("--lang=en", "completion", shell)
		_, b, _ := invoke("--lang=zh-CN", "completion", shell)
		if a != b {
			t.Fatalf("%s completion changed with language", shell)
		}
	}
}

func TestLocalizedInitStatusAndPrompt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "中文项目")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".node-version"), []byte("22.23.2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	call := func(args ...string) string {
		out.Reset()
		diag.Reset()
		code := Execute(append([]string{"--lang=zh-CN", "-C", dir}, args...), strings.NewReader(""), &out, &diag, "test")
		if code != 0 {
			t.Fatalf("%v: %d %s", args, code, &diag)
		}
		return out.String()
	}
	if got := call("init", "--no-input"); !strings.Contains(got, "已创建") {
		t.Fatal(got)
	}
	bare := call()
	if got := call("status"); got != bare || !strings.Contains(got, "尚未准备") || !strings.Contains(got, dir) {
		t.Fatal(got, bare)
	}
	a := call("--json")
	b := call("status", "--json")
	if a != b {
		t.Fatal(a, b)
	}
	out.Reset()
	tool, version, err := versionPrompt(strings.NewReader("node@22\n"), &out, true)("", "Select the runtime to change.")
	if err != nil || tool != "node" || version != "22" || !strings.Contains(out.String(), "请输入") {
		t.Fatal(tool, version, err, &out)
	}
}

func TestChineseHelpCoverage(t *testing.T) {
	// Capture every command via the same construction as production help.
	root := &cobra.Command{Use: "myenv"}
	var dir string
	var js bool
	addRun(root, &dir, &js, "")
	addHelp(root, &js)
	addShellInit(root, "", &js)
	root.AddCommand(cleanCommand(&dir, &js, ""))
	for _, c := range root.Commands() {
		if _, ok := chineseCommands[c.Name()]; !ok {
			t.Fatal(c.Name())
		}
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if _, ok := chineseFlags[f.Name]; !ok {
				t.Fatal(f.Name)
			}
		})
	}
}
