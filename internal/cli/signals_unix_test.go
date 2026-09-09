//go:build !windows

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"myenv/internal/state"
)

func TestOperationSignalCancellation(t *testing.T) {
	if root := os.Getenv("MYENV_SIGNAL_TEST_ROOT"); root != "" {
		os.Exit(Execute([]string{"-C", root, "clean", "--json"}, os.Stdin, os.Stdout, os.Stderr, "test"))
	}
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			work := filepath.Join(root, ".myenv")
			if err := os.Mkdir(work, 0700); err != nil {
				t.Fatal(err)
			}
			store, err := state.Open(context.Background(), filepath.Join(work, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			store.Close()
			unlock, err := state.LockWorkspace(context.Background(), filepath.Join(work, "modify.lock"))
			if err != nil {
				t.Fatal(err)
			}
			defer unlock()
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestOperationSignalCancellation$")
			cmd.Env = append(os.Environ(), "MYENV_SIGNAL_TEST_ROOT="+root)
			var diagnostic bytes.Buffer
			cmd.Stderr = &diagnostic
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
			// This prefix is emitted only after PersistentPreRun installed the
			// signal context. The held lock prevents cleanup from completing.
			prefix := make([]byte, len(`{"schema":1,"data":{"items":[`))
			if _, err = io.ReadFull(stdout, prefix); err != nil {
				t.Fatal(err)
			}
			if err = cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			tail, err := io.ReadAll(stdout)
			if err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			var r result
			if err = json.Unmarshal(append(prefix, tail...), &r); err != nil {
				t.Fatal(err, string(tail))
			}
			if cmd.ProcessState.ExitCode() != 130 || r.OK || r.Changed || r.Error == nil || r.Error.Code != "CANCELED" || diagnostic.Len() != 0 {
				t.Fatalf("signal=%v exit=%d result=%+v stderr=%s", sig, cmd.ProcessState.ExitCode(), r, &diagnostic)
			}
		})
	}
}
