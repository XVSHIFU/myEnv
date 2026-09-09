package core

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func measureMixedNoopMemory(t *testing.T, root, generationID string) {
	measureMixedMemory(t, root, generationID, false)
}

func measureMixedStatusMemory(t *testing.T, root, generationID string) {
	measureMixedMemory(t, root, generationID, true)
}

func measureMixedMemory(t *testing.T, root, generationID string, status bool) {
	t.Helper()
	output := os.Getenv("MYENV_TEST_MIXED_NOOP_MEMORY")
	if status {
		output = os.Getenv("MYENV_TEST_MIXED_STATUS_MEMORY")
	}
	if output == "" {
		return
	}
	if runtime.GOOS != "windows" {
		t.Fatal("mixed memory measurement requires Windows")
	}
	for _, name := range []string{"MYENV_TEST_MEMORY_SCRIPT", "MYENV_TEST_CLI_EXECUTABLE"} {
		if !filepath.IsAbs(os.Getenv(name)) {
			t.Fatalf("%s must be absolute", name)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	script := `& $env:MYENV_TEST_MEMORY_SCRIPT -Executable $env:MYENV_TEST_CLI_EXECUTABLE -CommandArguments @('-C', $env:MYENV_MEMORY_FIXTURE, 'sync', '--locked', '--no-input', '--json') -ExpectedGeneration $env:MYENV_MEMORY_GENERATION -Output $env:MYENV_TEST_MIXED_NOOP_MEMORY`
	if status {
		script = `& $env:MYENV_TEST_MEMORY_SCRIPT -Executable $env:MYENV_TEST_CLI_EXECUTABLE -CommandArguments @('-C', $env:MYENV_MEMORY_FIXTURE, '--json') -StatusResponse -ExpectedGeneration $env:MYENV_MEMORY_GENERATION -Output $env:MYENV_TEST_MIXED_STATUS_MEMORY`
	}
	command := exec.CommandContext(ctx, "pwsh", "-NoProfile", "-NonInteractive", "-Command", script)
	command.Env = append(os.Environ(), "MYENV_MEMORY_FIXTURE="+root, "MYENV_MEMORY_GENERATION="+generationID)
	if diagnostic, err := command.CombinedOutput(); err != nil {
		t.Fatalf("memory measurement: %v: %s", err, diagnostic)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Samples    []uint64 `json:"samples_peak_working_set_bytes"`
		Peak       float64  `json:"max_peak_working_set_bytes"`
		Generation string   `json:"expected_unchanged_generation"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Samples) != 20 || report.Peak <= 0 || report.Generation != generationID {
		t.Fatalf("incomplete report: %+v", report)
	}
	t.Logf("mixed CLI peak=%.4f MiB (status=%t); child memory excluded", report.Peak/(1<<20), status)
}
