package core

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
	"runtime"
	"sort"
	"testing"
	"time"
)

func measureMixedStatusCLI(t *testing.T, root, generationID string) {
	t.Helper()
	output := os.Getenv("MYENV_TEST_MIXED_STATUS_TIMINGS")
	if output == "" {
		return
	}
	executable := os.Getenv("MYENV_TEST_CLI_EXECUTABLE")
	if !filepath.IsAbs(executable) {
		t.Fatal("MYENV_TEST_CLI_EXECUTABLE must be absolute")
	}
	digest := func(path string) string {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			t.Fatal(err)
		}
		return hex.EncodeToString(hash.Sum(nil))
	}
	cliHash := digest(executable)
	database := filepath.Join(root, ".myenv", "state.db")
	stateHash := digest(database)
	args := []string{"-C", root, "--json"}
	samples := make([]float64, 0, 51)
	for i := 0; i < 51; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		command := exec.CommandContext(ctx, executable, args...)
		command.Dir = root
		var out, diagnostic bytes.Buffer
		command.Stdout, command.Stderr = &out, &diagnostic
		start := time.Now()
		err := command.Run()
		elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
		cancel()
		if err != nil {
			t.Fatalf("status: %v: %s", err, diagnostic.Bytes())
		}
		var response struct {
			OK      bool
			Changed bool
			Data    *Status
		}
		if err := json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if !response.OK || response.Changed || response.Data == nil || response.Data.Environment != "ready" || response.Data.Generation == nil || response.Data.Generation.ID != generationID {
			t.Fatalf("unexpected status: %s", out.Bytes())
		}
		samples = append(samples, elapsed)
	}
	if digest(database) != stateHash {
		t.Fatal("status modified database bytes")
	}
	ordered := append([]float64(nil), samples[1:]...)
	sort.Float64s(ordered)
	report := map[string]any{
		"cli": executable, "sha256": cliHash, "arguments": args, "platform": runtime.GOOS + "/" + runtime.GOARCH,
		"scope":             "Full CLI status in isolated real Node+Python fixture; includes native project input checks when configured; database bytes unchanged",
		"method":            "51 invocations; first separate, nearest-rank p95 of remaining 50; no cache eviction; fixture preparation excluded",
		"first_observed_ms": samples[0], "samples_ms": samples[1:], "median_ms": ordered[24], "p95_ms": ordered[47],
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Write(append(data, '\n'))
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
	t.Logf("full mixed status median=%.4fms p95=%.4fms", ordered[24], ordered[47])
}
