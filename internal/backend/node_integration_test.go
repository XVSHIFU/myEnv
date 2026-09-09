package backend

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"myenv/internal/runner"
)

// Explicit gate keeps large downloads and runtime execution out of unit tests.
func TestNodeOfficialPrepare(t *testing.T) {
	if os.Getenv("MYENV_TEST_PREPARE") != "1" {
		t.Skip("explicit real runtime preparation gate")
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	testNodeOfficialPrepare(t, platform)
}

// WSL compatibility evidence is separate from the product's supported target
// detection. This does not enable WSL installs through the CLI.
func TestNodeOfficialPrepareWSL(t *testing.T) {
	if os.Getenv("MYENV_TEST_PREPARE") != "1" {
		t.Skip("explicit real runtime preparation gate")
	}
	kernel, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || err != nil || !strings.Contains(strings.ToLower(string(kernel)), "microsoft") {
		t.Fatal("this compatibility test requires WSL Linux amd64")
	}
	if _, err := os.Stat("/lib64/ld-linux-x86-64.so.2"); err != nil {
		t.Fatal(err)
	}
	t.Log("WSL compatibility only; CLI target remains unsupported")
	testNodeOfficialPrepare(t, "linux-amd64-glibc")
}

func testNodeOfficialPrepare(t *testing.T, platform string) {
	t.Helper()
	root := os.Getenv("MYENV_TEST_ARTIFACTS")
	if root == "" {
		t.Fatal("MYENV_TEST_ARTIFACTS must name an isolated artifact directory")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	backend := NewNode()
	artifact, err := backend.Resolve(ctx, "22", platform)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := backend.Download(ctx, artifact, root)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "node-"+artifact.Version+"-"+time.Now().Format("20060102T150405.000000000"))
	var executable string
	if platform == "windows-amd64" {
		executable, err = PrepareNodeZIP(ctx, archive, destination, artifact)
	} else {
		executable, err = PrepareNodeTarGZ(ctx, archive, destination, artifact)
	}
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.CommandContext(ctx, executable, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("Node execution: %v %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "v"+artifact.Version {
		t.Fatalf("wrong runtime: %s", output)
	}
	npmCLI := filepath.Join(filepath.Dir(executable), "node_modules", "npm", "bin", "npm-cli.js")
	if runtime.GOOS != "windows" {
		npmCLI = filepath.Join(filepath.Dir(filepath.Dir(executable)), "lib", "node_modules", "npm", "bin", "npm-cli.js")
	}
	npmOutput, err := exec.CommandContext(ctx, executable, npmCLI, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("npm execution: %v %s", err, npmOutput)
	}
	if strings.TrimSpace(string(npmOutput)) != artifact.NPM {
		t.Fatalf("wrong npm: %s", npmOutput)
	}
	record := struct {
		Artifact            NodeArtifact
		Archive, Executable string
	}{artifact, archive, executable}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "prepared.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("verified Node %s and npm %s; retained %s", artifact.Version, artifact.NPM, executable)
}
