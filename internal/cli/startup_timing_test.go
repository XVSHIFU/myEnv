package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Use the same process launcher as the status/run measurements to distinguish
// command cost from differences between measurement hosts.
func TestMeasureInformationalCLI(t *testing.T) {
	output := os.Getenv("MYENV_TEST_INFORMATIONAL_TIMINGS")
	if output == "" {
		t.Skip("set MYENV_TEST_INFORMATIONAL_TIMINGS")
	}
	executable := os.Getenv("MYENV_TEST_CLI_EXECUTABLE")
	if !filepath.IsAbs(executable) {
		t.Fatal("CLI executable must be absolute")
	}
	binary, err := os.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, binary)
	closeErr := binary.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatal(copyErr, closeErr)
	}
	samples := map[string][]float64{"--help": {}, "--version": {}}
	root := t.TempDir()
	for i := 0; i < 51; i++ {
		flags := []string{"--version", "--help"}
		if i%2 != 0 {
			flags[0], flags[1] = flags[1], flags[0]
		}
		for _, flag := range flags {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			command := exec.CommandContext(ctx, executable, flag)
			command.Dir = root
			var out, diagnostic bytes.Buffer
			command.Stdout, command.Stderr = &out, &diagnostic
			start := time.Now()
			err := command.Run()
			elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
			cancel()
			if err != nil || diagnostic.Len() != 0 {
				t.Fatalf("%s: %v: %s", flag, err, diagnostic.Bytes())
			}
			if out.Len() == 0 || (flag == "--help" && !strings.Contains(out.String(), "Usage:") && !strings.Contains(out.String(), "用法：")) {
				t.Fatalf("unexpected %s output: %s", flag, out.Bytes())
			}
			samples[flag] = append(samples[flag], elapsed)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("informational command created project state: %v %v", entries, err)
	}
	report := map[string]any{"cli": executable, "sha256": hex.EncodeToString(hash.Sum(nil)), "method": "Go os/exec; 51 alternating paired invocations; first separate; nearest-rank p95 of remaining 50; no cache eviction"}
	for flag, values := range samples {
		ordered := append([]float64(nil), values[1:]...)
		sort.Float64s(ordered)
		report[flag] = map[string]any{"first_observed_ms": values[0], "samples_ms": values[1:], "median_ms": ordered[24], "p95_ms": ordered[47]}
		t.Logf("%s median=%gms p95=%gms", flag, ordered[24], ordered[47])
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Write(append(data, '\n'))
	closeErr = file.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
}
