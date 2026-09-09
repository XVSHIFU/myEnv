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

type firstFeedbackWriter struct {
	buffer    bytes.Buffer
	firstLine time.Time
}

func (w *firstFeedbackWriter) Write(p []byte) (int, error) {
	if w.firstLine.IsZero() && bytes.IndexByte(p, '\n') >= 0 {
		w.firstLine = time.Now()
	}
	return w.buffer.Write(p)
}

func (w *firstFeedbackWriter) Bytes() []byte { return w.buffer.Bytes() }

func measureMixedNoopCLI(t *testing.T, root, generationID string, pythonProject bool) {
	t.Helper()
	output := os.Getenv("MYENV_TEST_MIXED_NOOP_TIMINGS")
	if output == "" {
		return
	}
	executable := os.Getenv("MYENV_TEST_CLI_EXECUTABLE")
	if !filepath.IsAbs(executable) {
		t.Fatal("MYENV_TEST_CLI_EXECUTABLE must be absolute")
	}
	binary, err := os.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	_, err = io.Copy(hash, binary)
	binary.Close()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"-C", root, "sync", "--locked", "--no-input", "--json"}
	samples := make([]float64, 0, 51)
	feedback := make([]float64, 0, 51)
	for i := 0; i < 51; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		command := exec.CommandContext(ctx, executable, args...)
		command.Dir = root
		var out bytes.Buffer
		var diagnostic firstFeedbackWriter
		command.Stdout, command.Stderr = &out, &diagnostic
		start := time.Now()
		err = command.Run()
		elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
		cancel()
		if err != nil {
			t.Fatalf("full CLI sync: %v: %s", err, diagnostic.Bytes())
		}
		if diagnostic.firstLine.IsZero() || !bytes.HasPrefix(diagnostic.Bytes(), []byte("Checking environment...")) {
			t.Fatal("missing initial checking phase")
		}
		feedback = append(feedback, float64(diagnostic.firstLine.Sub(start).Nanoseconds())/1e6)
		var result struct {
			OK      bool       `json:"ok"`
			Changed bool       `json:"changed"`
			Data    SyncResult `json:"data"`
		}
		if err = json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !result.OK || result.Changed || result.Data.Changed || result.Data.LockChanged || result.Data.NativeLockChanged || result.Data.Generation == nil || result.Data.Generation.ID != generationID {
			t.Fatalf("measurement was not a no-op: %s", out.Bytes())
		}
		samples = append(samples, elapsed)
	}
	ordered := append([]float64(nil), samples[1:]...)
	sort.Float64s(ordered)
	orderedFeedback := append([]float64(nil), feedback[1:]...)
	sort.Float64s(orderedFeedback)
	maxFeedback := feedback[0]
	for _, sample := range feedback {
		if sample > maxFeedback {
			maxFeedback = sample
		}
	}
	scope := "Full external CLI no-op sync in an isolated real Node+Python runtime project without Python project dependencies; local Node source stopped before measurement"
	if pythonProject {
		scope = "Full external CLI no-op sync in an isolated Node+Python project with pyproject.toml, uv.lock and one installed pure-Python wheel; local artifact service stopped before measurement"
	}
	report := map[string]any{
		"cli": executable, "sha256": hex.EncodeToString(hash.Sum(nil)), "arguments": args,
		"platform": runtime.GOOS + "/" + runtime.GOARCH, "generation": generationID,
		"scope":             scope,
		"method":            "51 invocations, first separate, nearest-rank p95 of remaining 50; includes executable startup, excludes fixture preparation; no cache eviction",
		"first_observed_ms": samples[0], "samples_ms": samples[1:], "median_ms": ordered[24], "p95_ms": ordered[47],
		"initial_feedback": map[string]any{"samples_ms": feedback, "first_observed_ms": feedback[0], "hot_p95_ms": orderedFeedback[47], "max_ms": maxFeedback, "method": "From before exec.Run to receipt of the first complete stderr line; verifies Checking environment prefix; includes pipe delivery, not terminal rendering"},
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
	t.Logf("full mixed no-op sync median=%.4fms p95=%.4fms", ordered[24], ordered[47])
	t.Logf("initial feedback hot_p95=%.4fms max_all=%.4fms", orderedFeedback[47], maxFeedback)
}
