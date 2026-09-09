package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestDarwinPromptCancellationAndInput(t *testing.T) {
	if os.Getenv("MYENV_DARWIN_PROMPT_CHILD") != "1" {
		python, err := exec.LookPath("python3")
		if err != nil {
			t.Skip("python3 required to create an isolated native PTY")
		}
		const script = `import os, pty, subprocess, sys
master, slave = pty.openpty()
child = None
try:
    env = dict(os.environ, MYENV_DARWIN_PROMPT_CHILD='1')
    child = subprocess.Popen([sys.argv[1], '-test.run=^TestDarwinPromptCancellationAndInput$', '-test.timeout=5s'], stdin=slave, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
    line = child.stdout.readline()
    if line != b'ready-next\n':
        raise RuntimeError('missing input barrier: ' + repr(line))
    os.write(master, b'node@22\n')
    out, err = child.communicate(timeout=7)
    sys.stdout.buffer.write(out)
    sys.stderr.buffer.write(err)
    if child.returncode:
        raise RuntimeError('test child exit ' + str(child.returncode))
finally:
    if child is not None and child.poll() is None:
        child.kill()
        child.wait()
    os.close(master)
    os.close(slave)
`
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, python, "-c", script, os.Args[0])
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("native PTY: %v: %s", err, output)
		}
		t.Logf("%s", output)
		return
	}
	input := os.Stdin
	before, err := unix.IoctlGetTermios(int(input.Fd()), unix.TIOCGETA)
	if err != nil {
		t.Fatal(err)
	}
	flags, err := unix.FcntlInt(input.Fd(), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	buffer := make([]byte, 128)
	n, err := promptInput(ctx, input).Read(buffer)
	if n != 0 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled read: %d %v", n, err)
	}
	fmt.Fprintln(os.Stdout, "ready-next")
	next, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	n, err = promptInput(next, input).Read(buffer)
	if err != nil || string(buffer[:n]) != "node@22\n" {
		t.Fatalf("next input: %q %v", buffer[:n], err)
	}
	after, err := unix.IoctlGetTermios(int(input.Fd()), unix.TIOCGETA)
	if err != nil || *after != *before {
		t.Fatalf("terminal settings changed: %v", err)
	}
	afterFlags, err := unix.FcntlInt(input.Fd(), unix.F_GETFL, 0)
	if err != nil || flags != afterFlags {
		t.Fatalf("input flags changed: %d %d %v", flags, afterFlags, err)
	}
}
