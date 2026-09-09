package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWindowsNativeClosedStdioPreventsStart(t *testing.T) {
	root := t.TempDir()
	closed, err := os.Create(filepath.Join(root, "closed"))
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	command, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Fatal(err)
	}
	code, err := Execute(context.Background(), Process{Executable: command, Args: []string{"/d", "/c", "echo executed>marker"}, Directory: root, Stdout: closed})
	if code != 1 || err == nil || errors.Is(err, ErrTreeUnconfirmed) {
		t.Fatalf("closed stdout: %d %v", code, err)
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); !os.IsNotExist(err) {
		t.Fatalf("child ran with closed stdout: %v", err)
	}
}

func TestWindowsNativeMissingExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.exe")
	code, err := Execute(context.Background(), Process{Executable: path})
	if code != 1 || !errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrTreeUnconfirmed) {
		t.Fatalf("missing executable: %d %v", code, err)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != path {
		t.Fatalf("missing executable path in diagnostic: %v", err)
	}
}
