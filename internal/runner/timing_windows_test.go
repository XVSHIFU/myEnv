package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"sort"
	"testing"
	"time"
)

func TestMeasureWindowsSupervision(t *testing.T) {
	path := os.Getenv("MYENV_TEST_SUPERVISION_TIMINGS")
	if path == "" {
		t.Skip("requires dedicated timing report")
	}
	data, err := os.ReadFile(os.Getenv("MYENV_TEST_PREPARED_RECORD"))
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var samples [2][]float64
	for i := 0; i < 51; i++ {
		for offset := 0; offset < 2; offset++ {
			mode := (i + offset) % 2
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			var diagnostic bytes.Buffer
			p := Process{Executable: prepared.Executable, Args: []string{"-e", ""}, Directory: dir, Environment: os.Environ(), Stdin: bytes.NewReader(nil), Stdout: io.Discard, Stderr: &diagnostic}
			start := time.Now()
			if mode == 0 {
				cmd := exec.CommandContext(ctx, p.Executable, p.Args...)
				cmd.Dir, cmd.Env, cmd.Stdin, cmd.Stdout, cmd.Stderr = p.Directory, p.Environment, p.Stdin, p.Stdout, p.Stderr
				err = cmd.Run()
			} else {
				var code int
				code, err = Execute(ctx, p)
				if err == nil && code != 0 {
					t.Fatalf("runner exit %d", code)
				}
			}
			samples[mode] = append(samples[mode], float64(time.Since(start).Nanoseconds())/1e6)
			cancel()
			if err != nil {
				t.Fatalf("mode %d: %v %s", mode, err, diagnostic.Bytes())
			}
		}
	}
	delta := make([]float64, 50)
	for i := range delta {
		delta[i] = samples[1][i+1] - samples[0][i+1]
	}
	ordered := append([]float64(nil), delta...)
	sort.Float64s(ordered)
	report := map[string]any{"scope": "Windows in-process runner versus direct Node; same pipes; no CLI or lease transactions", "method": "51 rotated pairs, first excluded, 50 paired deltas; no cache eviction", "node": prepared.Executable, "direct_ms": samples[0], "runner_ms": samples[1], "paired_overhead_ms": delta, "median_ms": ordered[24], "p95_ms": ordered[47]}
	data, err = json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("report: %v %v", writeErr, closeErr)
	}
	t.Logf("supervision overhead median %.4f ms, p95 %.4f ms", ordered[24], ordered[47])
}
