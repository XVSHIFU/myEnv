package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCanceledReceiptFailureIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	defer write.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := Execute(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "printf x; sleep 30"}, Stdout: write, Completion: &TreeCompletion{File: file, Token: strings.Repeat("c", 64)}})
		write.Close()
		result <- err
	}()
	var ready [1]byte
	_, readyErr := io.ReadFull(read, ready[:])
	cancel()
	err = <-result
	if readyErr != nil || ready[0] != 'x' {
		t.Fatalf("child readiness: %v", readyErr)
	}
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, ErrTreeUnconfirmed) || !strings.Contains(err.Error(), "bad file descriptor") {
		t.Fatalf("receipt failure hidden by cancellation: %v", err)
	}
}

func TestLinuxSupervisorCompletionReceipt(t *testing.T) {
	for _, killed := range []bool{false, true} {
		name := "complete"
		if killed {
			name = "killed"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "receipt")
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			script := "if test -e /proc/self/fd/6; then exit 9; fi; exit 17"
			if killed {
				script = "kill -KILL $PPID"
			}
			completion := &TreeCompletion{File: file, Token: strings.Repeat("b", 64)}
			p := Process{Completion: completion, Executable: "/bin/sh", Args: []string{"-c", script}, Environment: os.Environ()}
			code, complete, err := executeLinuxSupervisor(context.Background(), p)
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if killed {
				if complete || err == nil || len(data) != 0 {
					t.Fatalf("dead supervisor manufactured receipt: %v %v %q", complete, err, data)
				}
				return
			}
			if !complete || err != nil || code != 17 || string(data) != "myenv-tree-complete-v1:"+completion.Token+"\n" {
				t.Fatalf("receipt: %d %v %v %q", code, complete, err, data)
			}
			// A receipt is single-use; a later launch must not inherit old proof.
			if _, _, err = executeLinuxSupervisor(context.Background(), p); err == nil {
				t.Fatal("reused completed receipt")
			}
		})
	}
}
