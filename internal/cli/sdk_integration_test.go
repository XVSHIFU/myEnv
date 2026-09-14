package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSDKRealTrial(t *testing.T) {
	root := os.Getenv("MYENV_SDK_TEST_ROOT")
	selection := os.Getenv("MYENV_SDK_TEST_SELECTION")
	if root == "" || selection == "" {
		t.Skip("requires explicit isolated SDK trial directory and selection")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("trial root must be absolute")
	}
	tool, version, ok := strings.Cut(selection, "@")
	if !ok {
		t.Fatal("invalid selection")
	}
	t.Setenv("GOCACHE", filepath.Join(root, "go-build-cache"))
	t.Setenv("GOPATH", filepath.Join(root, "go-workspace"))
	t.Setenv("CARGO_HOME", filepath.Join(root, "cargo-home"))
	project := filepath.Join(root, tool)
	namespace := filepath.Join(root, "namespace-"+tool)
	if err := os.MkdirAll(project, 0700); err != nil {
		t.Fatal(err)
	}
	declaration := filepath.Join(project, "myenv.yaml")
	if _, err := os.Stat(declaration); os.IsNotExist(err) {
		if err = os.WriteFile(declaration, []byte(fmt.Sprintf("schema: 1\ntools:\n  %s: %q\n", tool, version)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	call := func(args ...string) string {
		t.Helper()

		outFile, e := os.CreateTemp(project, "sdk-stdout-*")
		if e != nil {
			t.Fatal(e)
		}
		diagnosticFile, e := os.CreateTemp(project, "sdk-stderr-*")
		if e != nil {
			outFile.Close()
			t.Fatal(e)
		}
		args = append([]string{"-C", project, "--no-input"}, args...)
		code := execute(args, strings.NewReader(""), outFile, diagnosticFile, "sdk-candidate", namespace)
		outFile.Close()
		diagnosticFile.Close()
		out, e := os.ReadFile(outFile.Name())
		if e != nil {
			t.Fatal(e)
		}
		diagnostic, e := os.ReadFile(diagnosticFile.Name())
		if e != nil {
			t.Fatal(e)
		}
		if code != 0 {
			t.Fatalf("%v exit %d: %s %s", args, code, out, diagnostic)
		}
		t.Logf("%v: %s %s", args, out, diagnostic)
		return string(out)
	}
	call("use", selection)
	call("sync", "--locked")
	call("doctor", "--deep")
	switch tool {
	case "node":
		if !strings.Contains(call("run", "node", "-e", "console.log('myenv-sdk-ok')"), "myenv-sdk-ok") {
			t.Fatal("wrong Node output")
		}
	case "python":
		t.Setenv("PYTHONHOME", filepath.Join(root, "wrong-python-home"))
		t.Setenv("VIRTUAL_ENV", filepath.Join(root, "other-venv"))
		if !strings.Contains(call("run", "python", "-c", "import os,sys; assert os.path.normcase(os.path.realpath(sys.prefix)) == os.path.normcase(os.path.realpath(os.environ['VIRTUAL_ENV'])); assert 'PYTHONHOME' not in os.environ; print('python-isolation-ok')"), "python-isolation-ok") {
			t.Fatal("wrong Python isolation")
		}
		if !strings.Contains(call("run", "python", "-c", "import sys,ssl,sqlite3; print('myenv-sdk-ok'); print(sys.version); print(sys.base_prefix)"), "myenv-sdk-ok") {
			t.Fatal("wrong Python output")
		}
	case "java":
		os.WriteFile(filepath.Join(project, "Hello.java"), []byte("class Hello { public static void main(String[] args) { System.out.println(\"myenv-sdk-ok\"); } }"), 0600)
		call("run", "javac", "Hello.java")
		if !strings.Contains(call("run", "java", "Hello"), "myenv-sdk-ok") {
			t.Fatal("wrong Java output")
		}
	case "go":
		os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"myenv-sdk-ok\")}\n"), 0600)
		if !strings.Contains(call("run", "go", "run", "main.go"), "myenv-sdk-ok") {
			t.Fatal("wrong Go output")
		}
	case "rust":
		os.WriteFile(filepath.Join(project, "main.rs"), []byte("fn main(){println!(\"myenv-sdk-ok\");}\n"), 0600)
		call("run", "cargo", "--version")
		exe := "hello-sdk"
		if runtime.GOOS == "windows" {
			exe = "hello-sdk.exe"
		}
		call("run", "rustc", "main.rs", "-o", exe)
		if !strings.Contains(call("run", filepath.Join(project, exe)), "myenv-sdk-ok") {
			t.Fatal("wrong Rust output")
		}
		if err := os.WriteFile(filepath.Join(project, "Cargo.toml"), []byte("[package]\nname='myenv-sdk-trial'\nversion='0.1.0'\nedition='2021'\n[[bin]]\nname='myenv-sdk-trial'\npath='main.rs'\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(call("run", "cargo", "run", "--offline", "--quiet"), "myenv-sdk-ok") {
			t.Fatal("wrong Cargo output")
		}
	}
}
