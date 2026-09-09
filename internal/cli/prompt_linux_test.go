package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestLinuxPromptCancellationAndInput(t *testing.T) {
	master, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(master)
	if err = unix.IoctlSetPointerInt(master, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(master, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(fmt.Sprintf("/dev/pts/%d", number))
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	flags, err := unix.FcntlInt(input.Fd(), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	before, err := unix.IoctlGetTermios(int(input.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	b := make([]byte, 128)
	n, err := promptInput(ctx, input).Read(b)
	if n != 0 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancel %d %v", n, err)
	}
	if _, err = unix.Write(master, []byte("node@22\n")); err != nil {
		t.Fatal(err)
	}
	next, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	n, err = promptInput(next, input).Read(b)
	if err != nil || string(b[:n]) != "node@22\n" {
		t.Fatalf("input %q %v", b[:n], err)
	}
	afterFlags, err := unix.FcntlInt(input.Fd(), unix.F_GETFL, 0)
	if err != nil || afterFlags != flags {
		t.Fatalf("caller flags changed: %d %d %v", flags, afterFlags, err)
	}
	after, err := unix.IoctlGetTermios(int(input.Fd()), unix.TCGETS)
	if err != nil || *after != *before {
		t.Fatalf("terminal settings changed: %v", err)
	}
}

func TestLinuxUsePromptSignal(t *testing.T)  { linuxPromptSignal(t, "use", syscall.SIGINT) }
func TestLinuxUsePromptTerm(t *testing.T)    { linuxPromptSignal(t, "use", syscall.SIGTERM) }
func TestLinuxUsePromptHangup(t *testing.T)  { linuxPromptSignal(t, "use", syscall.SIGHUP) }
func TestLinuxInitPromptSignal(t *testing.T) { linuxPromptSignal(t, "init", syscall.SIGINT) }
func linuxPromptSignal(t *testing.T, command string, signal syscall.Signal) {
	t.Helper()
	if root := os.Getenv("MYENV_PROMPT_SIGNAL_ROOT"); root != "" {
		os.Unsetenv("CI")
		os.Exit(execute([]string{"-C", root, command}, os.Stdin, os.Stdout, os.Stderr, "test", root))
	}
	master, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(master)
	if err = unix.IoctlSetPointerInt(master, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(master, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	before, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), "MYENV_PROMPT_SIGNAL_ROOT="+root)
	cmd.Stdin, cmd.Stdout = slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	diagnostic, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	suffix := []byte("Enter tool@version (python or node): ")
	var prompt bytes.Buffer
	for !bytes.HasSuffix(prompt.Bytes(), suffix) && prompt.Len() < 4096 {
		var one [1]byte
		if _, err = io.ReadFull(diagnostic, one[:]); err != nil {
			t.Fatalf("prompt %q: %v", prompt.String(), err)
		}
		prompt.WriteByte(one[0])
	}
	if !bytes.HasSuffix(prompt.Bytes(), suffix) {
		t.Fatalf("prompt too long: %q", prompt.String())
	}
	if err = cmd.Process.Signal(signal); err != nil {
		t.Fatal(err)
	}
	remaining, readErr := io.ReadAll(io.LimitReader(diagnostic, 8192))
	err = cmd.Wait()
	var exit *exec.ExitError
	if readErr != nil || !errors.As(err, &exit) || exit.ExitCode() != 130 || !strings.Contains(string(remaining), "CANCELED") {
		t.Fatalf("signal result: %v %v %s", err, readErr, remaining)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("prompt created state: %v %v", entries, err)
	}
	after, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil || *before != *after {
		t.Fatalf("terminal changed: %v", err)
	}
}
