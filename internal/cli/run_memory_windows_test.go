package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMeasureFullRunMemory(t *testing.T) {
	output := os.Getenv("MYENV_TEST_RUN_MEMORY_REPORT")
	if output == "" {
		t.Skip("requires a new memory report path, measurement script and CLI executable")
	}
	for _, name := range []string{"MYENV_TEST_MEMORY_SCRIPT", "MYENV_TEST_CLI_EXECUTABLE"} {
		path := os.Getenv(name)
		if !filepath.IsAbs(path) {
			t.Fatalf("%s must be an absolute path", name)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	root, prepared := prepareRetainedNodeRun(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	// Values remain data in environment variables; no user path is shell code.
	command := exec.CommandContext(ctx, "pwsh", "-NoProfile", "-NonInteractive", "-Command",
		`& $env:MYENV_TEST_MEMORY_SCRIPT -Executable $env:MYENV_TEST_CLI_EXECUTABLE -CommandArguments @('-C', $env:MYENV_MEMORY_FIXTURE, 'run', 'node', '-e', '') -Output $env:MYENV_TEST_RUN_MEMORY_REPORT`)
	command.Env = append(os.Environ(), "MYENV_MEMORY_FIXTURE="+root)
	if diagnostic, err := command.CombinedOutput(); err != nil {
		t.Fatalf("memory measurement: %v: %s", err, diagnostic)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Samples []uint64 `json:"samples_peak_working_set_bytes"`
		Peak    float64  `json:"max_peak_working_set_bytes"`
		SHA256  string   `json:"sha256"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Samples) != 20 || report.Peak == 0 || len(report.SHA256) != 64 {
		t.Fatalf("incomplete memory report: %+v", report)
	}
	t.Logf("full CLI run: peak=%.4f MiB sha256=%s; retained Node=%s; child memory excluded", float64(report.Peak)/(1<<20), report.SHA256, prepared.Artifact.Version)
}
