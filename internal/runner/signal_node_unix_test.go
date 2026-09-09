//go:build !windows

package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestForwardSignalRetainedNode(t *testing.T) {
	testForwardSignalNode(t, Execute)
}

func testForwardSignalNode(t *testing.T, execute func(context.Context, Process) (int, error)) {
	testNodeSignalDelivery(t, execute, false)
}

func testNodeSignalDelivery(t *testing.T, execute func(context.Context, Process) (int, error), group bool) {
	testNodeSignalDeliveryWithSignal(t, execute, group, syscall.SIGTERM, "SIGTERM")
}

func testNodeSignalDeliveryWithSignal(t *testing.T, execute func(context.Context, Process) (int, error), group bool, sig syscall.Signal, signalName string) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	if os.Getenv("MYENV_NODE_SIGNAL_HELPER") == "1" {
		data, err := os.ReadFile(record)
		if err != nil {
			t.Fatal(err)
		}
		var prepared struct{ Executable string }
		if err = json.Unmarshal(data, &prepared); err != nil {
			t.Fatal(err)
		}
		script := "process.on('SIGTERM',()=>process.exit(23)); process.stdout.write('ready\\n'); setInterval(()=>{},1000)"
		if group {
			script = "let count=0; process.on('SIGTERM',()=>{if(++count===1)setTimeout(()=>process.exit(count===1?23:90+count),100)}); process.stdout.write('ready\\n'); setInterval(()=>{},1000)"
		}
		script = strings.ReplaceAll(script, "SIGTERM", signalName)
		if signalName == "SIGCONT" {
			// Fixed-width PID allows the parent to observe the actual stopped
			// state before sending CONT; readiness alone would race SIGSTOP.
			script += ";process.stdout.write(String(process.pid).padStart(10,'0'));process.kill(process.pid,'SIGSTOP')"
		}
		code, err := execute(context.Background(), Process{Executable: prepared.Executable, Args: []string{"-e", script}, Environment: os.Environ(), Stdout: os.Stdout, Stderr: os.Stderr})
		if err != nil {
			t.Fatal(err)
		}
		os.Exit(code)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), "MYENV_NODE_SIGNAL_HELPER=1")
	if group {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
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
	ready := make([]byte, 6)
	if _, err = io.ReadFull(stdout, ready); err != nil || string(ready) != "ready\n" {
		t.Fatalf("readiness: %q %v", ready, err)
	}
	if signalName == "SIGCONT" {
		pidBytes := make([]byte, 10)
		if _, err := io.ReadFull(stdout, pidBytes); err != nil {
			t.Fatal(err)
		}
		pid, err := strconv.Atoi(string(pidBytes))
		if err != nil || pid <= 1 {
			t.Fatalf("invalid fixture PID %q: %v", pidBytes, err)
		}
		for {
			data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
			if err != nil {
				t.Fatal(err)
			}
			end := strings.LastIndexByte(string(data), ')')
			if end < 0 {
				t.Fatal("invalid fixture process stat")
			}
			fields := strings.Fields(string(data[end+1:]))
			if len(fields) > 0 && fields[0] == "T" {
				break
			}
			if ctx.Err() != nil {
				t.Fatal("fixture did not stop", ctx.Err())
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	if group {
		err = syscall.Kill(-cmd.Process.Pid, sig)
	} else {
		err = cmd.Process.Signal(sig)
	}
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, stdout)
	_ = cmd.Wait()
	if cmd.ProcessState.ExitCode() != 23 || diagnostic.Len() != 0 {
		t.Fatalf("exit=%d stderr=%s", cmd.ProcessState.ExitCode(), &diagnostic)
	}
}
