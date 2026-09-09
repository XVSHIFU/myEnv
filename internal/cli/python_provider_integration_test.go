package cli

import (
	"bytes"
	"myenv/internal/config"
	"myenv/internal/runner"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonProviderSwitch(t *testing.T) {
	root := os.Getenv("MYENV_SDK_TEST_ROOT")
	if root == "" {
		t.Skip("explicit isolated trial required")
	}
	project := filepath.Join(root, "python")
	namespace := filepath.Join(root, "namespace-python")
	call := func(ok bool, args ...string) string {
		t.Helper()
		var out, diag bytes.Buffer
		code := execute(append([]string{"-C", project, "--no-input"}, args...), strings.NewReader(""), &out, &diag, "test", namespace)
		if (code == 0) != ok {
			t.Fatalf("%v code %d: %s %s", args, code, &out, &diag)
		}
		t.Logf("%v exit %d", args, code)
		return out.String()
	}
	call(true, "use", "python@3.14", "--provider", "astral")
	platform, e := runner.Platform()
	if e != nil {
		t.Fatal(e)
	}
	lock, e := config.ReadLock(filepath.Join(project, "myenv.lock"))
	if e != nil || !strings.HasPrefix(lock.Platforms[platform].Tools["python"].Backend, "uv-") {
		t.Fatal(lock, e)
	}
	call(false, "use", "python@99.99.99", "--provider", "python.org")
	if !strings.Contains(call(true, "run", "--current", "python", "-c", "print('old-environment-ok')"), "old-environment-ok") {
		t.Fatal("old environment lost")
	}
	call(true, "use", "python@3.14", "--provider", "python.org")
	lock, e = config.ReadLock(filepath.Join(project, "myenv.lock"))
	if e != nil || lock.Platforms[platform].Tools["python"].Backend != "python-official-v1" {
		t.Fatal(lock, e)
	}
	call(true, "sync", "--locked")
}
