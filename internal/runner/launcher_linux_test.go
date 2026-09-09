package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLinuxSupervisorSignalNode(t *testing.T) {
	testForwardSignalNode(t, func(ctx context.Context, p Process) (int, error) {
		code, complete, err := executeLinuxSupervisor(ctx, p)
		if !complete {
			t.Fatal("tree completion missing", err)
		}
		return code, err
	})
}

func TestLinuxGroupSignalRetainedNode(t *testing.T) {
	testNodeSignalDelivery(t, Execute, true)
}

func TestLinuxSupervisorLifetimeClosedBeforeHandshake(t *testing.T) {
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer requestRead.Close()
	defer requestWrite.Close()
	resultRead, resultWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer resultRead.Close()
	defer resultWrite.Close()
	lifetimeRead, lifetimeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer lifetimeRead.Close()
	// The parent is already gone before the helper has registered anything.
	if err = lifetimeWrite.Close(); err != nil {
		t.Fatal(err)
	}
	if err = json.NewEncoder(requestWrite).Encode(supervisorRequest{Executable: "/bin/sh", Args: []string{"-c", "sleep 2"}, Environment: os.Environ()}); err != nil {
		t.Fatal(err)
	}
	requestWrite.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], supervisorArgument)
	command.ExtraFiles = []*os.File{requestRead, resultWrite, lifetimeRead}
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
	requestRead.Close()
	resultWrite.Close()
	lifetimeRead.Close()
	decoder := json.NewDecoder(resultRead)
	var ready, reply supervisorReply
	if err = decoder.Decode(&ready); err != nil || !ready.Ready {
		t.Fatal("readiness", ready, err)
	}
	if err = decoder.Decode(&reply); err != nil || !reply.Complete || !reply.Canceled {
		t.Fatal("early lifetime EOF not handled", reply, err)
	}
	if err = command.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestLinuxSupervisorTransport(t *testing.T) {
	root := t.TempDir()
	var out, diagnostic bytes.Buffer
	code, complete, err := executeLinuxSupervisor(context.Background(), Process{Executable: "/bin/sh", Args: []string{"-c", "cat; (sleep 0.1; printf done > finished) </dev/null >/dev/null 2>&1 & exit 17"}, Directory: root, Environment: os.Environ(), Stdin: strings.NewReader("input"), Stdout: &out, Stderr: &diagnostic})
	if err != nil || !complete || code != 17 || out.String() != "input" || diagnostic.Len() != 0 {
		t.Fatalf("code=%d complete=%v err=%v stdout=%q stderr=%q", code, complete, err, &out, &diagnostic)
	}
	if _, err := os.Stat(filepath.Join(root, "finished")); err != nil {
		t.Fatal("transport returned before descendant", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, complete, err = executeLinuxSupervisor(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "setsid sleep 20 </dev/null >/dev/null 2>&1 & wait"}, Environment: os.Environ()})
	if !complete || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel complete=%v err=%v", complete, err)
	}
}

func TestLinuxSupervisorFailureProof(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, complete, err := executeLinuxSupervisor(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "kill -KILL $PPID"}, Environment: os.Environ()})
	if err == nil || complete {
		t.Fatalf("dead supervisor reported safe completion: %v %v", complete, err)
	}
	_, complete, err = executeLinuxSupervisor(ctx, Process{Executable: "/bin/sh", Args: []string{strings.Repeat("x", 1<<20)}})
	if err == nil || !complete {
		t.Fatalf("oversized request: %v %v", complete, err)
	}
}

func TestExecuteLinuxUnconfirmedTree(t *testing.T) {
	_, err := Execute(context.Background(), Process{Executable: "/bin/sh", Args: []string{"-c", "kill -KILL $PPID"}, Environment: os.Environ()})
	if !errors.Is(err, ErrTreeUnconfirmed) {
		t.Fatalf("lost lease retention signal: %v", err)
	}
}
