package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"myenv/internal/core"
	"myenv/internal/runner"
)

func TestMeasureFullRunCLI(t *testing.T) {
	output := os.Getenv("MYENV_TEST_RUN_CLI_TIMINGS")
	if output == "" {
		t.Skip("set dedicated report and CLI executable paths to collect full run timings")
	}
	executable, err := filepath.Abs(os.Getenv("MYENV_TEST_CLI_EXECUTABLE"))
	if err != nil {
		t.Fatal(err)
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
	root, prepared := prepareRetainedNodeRun(t)
	env := os.Environ()
	args := []string{"-C", root, "run", "node", "-e", ""}
	invoke := func(ctx context.Context, path string, args []string) error {
		command := exec.CommandContext(ctx, path, args...)
		command.Dir, command.Env = root, env
		var diagnostic bytes.Buffer
		command.Stdout, command.Stderr = io.Discard, &diagnostic
		if err := command.Run(); err != nil {
			return fmt.Errorf("%w: %s", err, diagnostic.Bytes())
		}
		return nil
	}
	methods := []func(context.Context) error{
		func(ctx context.Context) error { return invoke(ctx, prepared.Executable, []string{"-e", ""}) },
		func(ctx context.Context) error { return invoke(ctx, executable, args) },
		func(ctx context.Context) error {
			var diagnostic bytes.Buffer
			code := ExecuteContext(ctx, args, bytes.NewReader(nil), io.Discard, &diagnostic, "measurement")
			if code != 0 {
				return fmt.Errorf("in-process CLI exit %d: %s", code, diagnostic.Bytes())
			}
			return nil
		},
	}
	names := []string{"direct_node", "full_cli", "in_process_cli"}
	// Build-flag comparisons only change the external CLI artifact.
	if os.Getenv("MYENV_TEST_RUN_CLI_ONLY") == "1" {
		methods, names = methods[:2], names[:2]
	}
	samples := make([][]float64, len(methods))
	for i := 0; i < 51; i++ {
		for offset := range methods {
			index := (offset + i) % len(methods)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			start := time.Now()
			err := methods[index](ctx)
			elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
			cancel()
			if err != nil {
				t.Fatal(names[index], err)
			}
			samples[index] = append(samples[index], elapsed)
		}
	}
	summarize := func(samples []float64) map[string]any {
		ordered := append([]float64(nil), samples...)
		sort.Float64s(ordered)
		return map[string]any{"samples_ms": samples, "median_ms": ordered[24], "p95_ms": ordered[47]}
	}
	results := map[string]any{}
	for i, name := range names {
		result := summarize(samples[i][1:])
		result["first_observed_ms"] = samples[i][0]
		if i != 0 {
			delta := make([]float64, 50)
			for j := range delta {
				delta[j] = samples[i][j+1] - samples[0][j+1]
			}
			result["paired_overhead"] = summarize(delta)
			t.Logf("%s overhead median_ms=%.4f p95_ms=%.4f", name, result["paired_overhead"].(map[string]any)["median_ms"], result["paired_overhead"].(map[string]any)["p95_ms"])
		}
		results[name] = result
	}
	report := map[string]any{"scope": "full CLI run with isolated small Node fixture; in-process comparison excludes executable initialization", "method": "51 samples per method, rotated ordering, first excluded, nearest-rank p95, paired direct Node baseline, no cache eviction", "cli": executable, "cli_sha256": hex.EncodeToString(hash.Sum(nil)), "node": prepared.Executable, "artifact": prepared.Artifact, "platform": runtime.GOOS + "/" + runtime.GOARCH, "go_version": runtime.Version(), "fixture": root, "results": results}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Write(append(data, '\n'))
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func TestMeasureRunStages(t *testing.T) {
	output := os.Getenv("MYENV_TEST_RUN_STAGE_TIMINGS")
	if output == "" {
		t.Skip("set dedicated report path to collect run stage timings")
	}
	root, prepared := prepareRetainedNodeRun(t)
	env, err := runner.Environment(os.Environ(), nil, []string{filepath.Dir(prepared.Executable)}, runtime.GOOS == "windows")
	if err != nil {
		t.Fatal(err)
	}
	var selection, execution, release []float64
	for i := 0; i < 51; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		start := time.Now()
		selected, err := (&core.Service{}).SelectRun(ctx, root, false)
		selection = append(selection, float64(time.Since(start).Nanoseconds())/1e6)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		start = time.Now()
		var diagnostic bytes.Buffer
		code, runErr := runner.Execute(ctx, runner.Process{TreeID: selected.TreeID, Completion: selected.Completion, Executable: prepared.Executable, Args: []string{"-e", ""}, Directory: root, Environment: env, Stdin: bytes.NewReader(nil), Stdout: io.Discard, Stderr: &diagnostic})
		execution = append(execution, float64(time.Since(start).Nanoseconds())/1e6)
		start = time.Now()
		err = selected.Finish(runErr)
		release = append(release, float64(time.Since(start).Nanoseconds())/1e6)
		cancel()
		if err != nil || runErr != nil || code != 0 {
			t.Fatalf("run: %d %v %v", code, runErr, err)
		}
	}
	results := map[string]any{}
	for name, values := range map[string][]float64{"selection": selection, "execution_including_node": execution, "release": release} {
		ordered := append([]float64(nil), values[1:]...)
		sort.Float64s(ordered)
		results[name] = map[string]any{"first_observed_ms": values[0], "samples_ms": values[1:], "median_ms": ordered[24], "p95_ms": ordered[47]}
		t.Logf("%s median_ms=%.4f p95_ms=%.4f", name, ordered[24], ordered[47])
	}
	data, err := json.MarshalIndent(map[string]any{"scope": "in-process stage diagnosis, execution includes user Node runtime", "method": "51 runs, first excluded, nearest-rank percentiles; not additive p95; empty reader stdin, discarded stdout, buffered stderr matching in-process CLI", "platform": runtime.GOOS + "/" + runtime.GOARCH, "go_version": runtime.Version(), "artifact": prepared.Artifact, "results": results}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Write(append(data, '\n'))
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
