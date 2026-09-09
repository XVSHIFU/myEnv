package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestShellInitFailures(t *testing.T) {
	for _, check := range []struct {
		name string
		args []string
		code int
		json bool
	}{
		{"missing-profile", []string{"shell-init", "bash"}, 1, false},
		{"unknown-shell", []string{"shell-init", "unknown"}, 2, false},
		{"missing-shell", []string{"shell-init"}, 2, false},
		{"extra-argument", []string{"shell-init", "bash", "extra"}, 2, false},
		{"json", []string{"shell-init", "bash", "--json"}, 2, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			namespace := t.TempDir()
			var out, diagnostic bytes.Buffer
			if code := execute(check.args, bytes.NewReader(nil), &out, &diagnostic, "test", namespace); code != check.code {
				t.Fatalf("exit %d: %s %s", code, out.String(), diagnostic.String())
			}
			if check.json {
				var response result
				if err := json.Unmarshal(out.Bytes(), &response); err != nil || response.OK || response.Changed || response.Error == nil || response.Error.Code != "USAGE_ERROR" || diagnostic.Len() != 0 {
					t.Fatalf("JSON failure: %s %s %v", out.String(), diagnostic.String(), err)
				}
			} else if out.Len() != 0 || diagnostic.Len() == 0 {
				t.Fatalf("partial script on failure: %s %s", out.String(), diagnostic.String())
			}
			if check.name == "missing-profile" && !strings.Contains(diagnostic.String(), "myenv use --global") {
				t.Fatal("missing profile action absent")
			}
			entries, err := os.ReadDir(namespace)
			if err != nil || len(entries) != 0 {
				t.Fatalf("failure created state: %v %v", entries, err)
			}
		})
	}
}

func TestShellInitProfile(t *testing.T) {
	namespace := t.TempDir()
	directory := filepath.Join(namespace, "myenv")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "profile.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		var out, diagnostic bytes.Buffer
		if code := execute([]string{"shell-init", shell}, bytes.NewReader(nil), &out, &diagnostic, "test", namespace); code != 0 || !strings.Contains(out.String(), "run --global node") || strings.Contains(out.String(), "run --global python") {
			t.Fatalf("%s: %d %s %s", shell, code, out.String(), diagnostic.String())
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("shell initialization wrote state: %v %v", entries, err)
	}
}

func TestPowerShellProfileWrapperArguments(t *testing.T) {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("requires PowerShell")
	}
	root := t.TempDir()
	entry := filepath.Join(root, "probe '$ literal.ps1")
	if err := os.WriteFile(entry, []byte("ConvertTo-Json -InputObject @($args) -Compress\n"), 0600); err != nil {
		t.Fatal(err)
	}
	driver := filepath.Join(root, "driver.ps1")
	script := profileShellScript("powershell", entry, map[string]string{"node": "22"}) + "node 'a b' '' '$literal;not-code'\n"
	if err := os.WriteFile(driver, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-File", driver).CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper: %v %s", err, output)
	}
	var args []string
	if err := json.Unmarshal(bytes.TrimSpace(output), &args); err != nil {
		t.Fatalf("arguments: %v %s", err, output)
	}
	want := []string{"run", "--global", "node", "a b", "", "$literal;not-code"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args=%q want=%q", args, want)
	}
}
