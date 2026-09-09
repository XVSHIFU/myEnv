package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLinuxSupervisorParentDeath(t *testing.T) {
	if root := os.Getenv("MYENV_PARENT_DEATH_HELPER"); root != "" {
		receipt, err := os.OpenFile(filepath.Join(root, "completion"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		defer receipt.Close()
		code, err := Execute(context.Background(), Process{
			Completion: &TreeCompletion{File: receipt, Token: strings.Repeat("a", 64)},
			Executable: "/bin/sh", Directory: root, Environment: os.Environ(),
			Args: []string{"-c", "echo $$ > leader; setsid /bin/sh -c 'echo $$ > descendant; exec sleep 5' </dev/null >/dev/null 2>&1 & wait"},
		})
		if err != nil {
			t.Fatal(err)
		}
		os.Exit(code)
	}
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLinuxSupervisorParentDeath$")
	command.Env = append(os.Environ(), "MYENV_PARENT_DEATH_HELPER="+root)
	started := time.Now()
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	var paths []string
	for _, name := range []string{"leader", "descendant"} {
		var pid int
		for time.Since(started) < time.Second {
			data, err := os.ReadFile(filepath.Join(root, name))
			if err == nil {
				pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
			}
			if pid > 1 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if pid <= 1 {
			t.Fatal("test child did not start promptly", name)
		}
		path := filepath.Join("/proc", strconv.Itoa(pid))
		if _, err := os.Stat(path); err != nil {
			t.Fatal("child already exited", err)
		}
		paths = append(paths, path)
	}
	if info, err := os.Stat(filepath.Join(root, "completion")); err != nil || info.Size() != 0 {
		t.Fatal("completion appeared while descendants were alive", err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("caller was not killed")
	}
	waited = true
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		remaining := false
		for _, path := range paths {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				remaining = true
			}
		}
		if !remaining {
			data, err := os.ReadFile(filepath.Join(root, "completion"))
			if err == nil && string(data) == "myenv-tree-complete-v1:"+strings.Repeat("a", 64)+"\n" {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	// The fixture expires on its own even if supervision fails. Avoid signaling
	// numeric PIDs after losing their ownership, and let it finish before cleanup.
	for time.Since(started) < 6*time.Second {
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("caller death did not reap the tree and persist its completion within one second")
}
