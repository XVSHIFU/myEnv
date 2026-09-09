package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"myenv/internal/runner"
	"myenv/internal/state"
)

// This measures components inside an already started process, not the full CLI
// p95 gate: config checks, executable loading and CLI initialization are absent.
func TestMeasureLinuxRunComponents(t *testing.T) {
	output := os.Getenv("MYENV_TEST_RUN_TIMINGS")
	if output == "" {
		t.Skip("set dedicated output path to collect run component timings")
	}
	root := t.TempDir()
	database := filepath.Join(root, "state.db")
	ctx := context.Background()
	store, err := state.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Publish(ctx, state.Generation{ID: "fixture", Directory: root, InputDigest: "fixture", NodeExecutable: "/bin/true"}, ""); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	env := os.Environ()
	direct := func(ctx context.Context) error {
		command := exec.CommandContext(ctx, "/bin/true")
		command.Dir, command.Env = root, env
		return command.Run()
	}
	supervised := func(ctx context.Context) error {
		code, err := runner.Execute(ctx, runner.Process{Executable: "/bin/true", Directory: root, Environment: env})
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("child exit %d", code)
		}
		return nil
	}
	leased := func(ctx context.Context) error {
		store, err := state.Open(ctx, database)
		if err != nil {
			return err
		}
		lease, err := acquireRunLease(ctx, store)
		if err != nil {
			store.Close()
			return err
		}
		selected := leasedRunEnvironment(store, lease)
		if err = prepareRunCompletion(ctx, root, store, selected); err != nil {
			selected.Release()
			return err
		}
		code, runErr := runner.Execute(ctx, runner.Process{Completion: selected.Completion, TreeID: selected.TreeID, Executable: "/bin/true", Directory: root, Environment: env})
		if err = selected.Finish(runErr); err != nil {
			return err
		}
		if runErr != nil {
			return runErr
		}
		if code != 0 {
			return fmt.Errorf("child exit %d", code)
		}
		return nil
	}
	names := []string{"direct", "supervisor", "lease_and_supervisor"}
	methods := []func(context.Context) error{direct, supervised, leased}
	samples := make([][]float64, len(methods))
	for i := 0; i < 51; i++ {
		// Rotate execution order to reduce a fixed ordering's cache/scheduling bias.
		for offset := range methods {
			index := (offset + i) % len(methods)
			callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			start := time.Now()
			err := methods[index](callCtx)
			elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
			cancel()
			if err != nil {
				t.Fatal(names[index], err)
			}
			samples[index] = append(samples[index], elapsed)
		}
	}
	summarize := func(values []float64) map[string]any {
		ordered := append([]float64(nil), values...)
		sort.Float64s(ordered)
		return map[string]any{"samples_ms": values, "median_ms": ordered[24], "p95_ms": ordered[47]}
	}
	results := map[string]any{}
	for i, name := range names {
		summary := summarize(samples[i][1:])
		summary["first_observed_ms"] = samples[i][0]
		results[name] = summary
		t.Logf("%s median_ms=%.4f p95_ms=%.4f", name, summary["median_ms"], summary["p95_ms"])
		if i != 0 {
			delta := make([]float64, 50)
			for n := range delta {
				delta[n] = samples[i][n+1] - samples[0][n+1]
			}
			summary["paired_overhead"] = summarize(delta)
		}
	}
	executable, err := os.Executable()
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
	var binaryFS, workFS unix.Statfs_t
	if err = unix.Statfs(executable, &binaryFS); err != nil {
		t.Fatal(err)
	}
	if err = unix.Statfs(root, &workFS); err != nil {
		t.Fatal(err)
	}
	kernel, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"scope": "in-process Linux components; not full CLI gate", "method": "51 samples per case, rotated order, first excluded, nearest-rank p95, no cache eviction", "executable": executable, "sha256": hex.EncodeToString(hash.Sum(nil)), "binary_fs_type": fmt.Sprintf("0x%x", binaryFS.Type), "work_fs_type": fmt.Sprintf("0x%x", workFS.Type), "kernel": strings.TrimSpace(string(kernel)), "results": results}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
