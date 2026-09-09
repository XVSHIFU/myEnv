package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Process struct {
	Completion  *TreeCompletion
	TreeID      string
	Executable  string
	Args        []string
	Directory   string
	Environment []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

// TreeCompletion is an exclusive, initially empty regular file plus a random
// binding token owned by the caller. Linux can persist proof after parent death.
// The caller owns closing File and registering Token before execution begins.
type TreeCompletion struct {
	File  *os.File
	Token string
}

type supervision struct {
	Start  func() error
	Finish func() error
	Close  func()
}

// ErrTreeUnconfirmed means callers must retain the generation lease.
var ErrTreeUnconfirmed = errors.New("process tree completion is unconfirmed")

// The supervisor confirmed completion after interrupting remaining descendants.
// This is cancellation with proof, not a reason to retain an uncertain lease.
var errTreeInterrupted = errors.New("remaining process tree interrupted")

// Execute waits for the child before returning, keeping the caller's lease alive.
// No shell is involved; executable resolution is explicit and precedes this call.
func Execute(ctx context.Context, p Process) (int, error) {
	if err := ctx.Err(); err != nil {
		return 1, err
	}
	if !filepath.IsAbs(p.Executable) {
		return 1, fmt.Errorf("executable must be an absolute path")
	}
	return executePlatform(ctx, p)
}

func executeDirect(ctx context.Context, p Process) (int, error) {
	return executeDirectWithPreparation(ctx, p, prepareSupervision)
}

func executeDirectWithPreparation(ctx context.Context, p Process, prepare func(context.Context, *exec.Cmd, string) (*supervision, error)) (int, error) {
	command := exec.CommandContext(ctx, p.Executable, p.Args...)
	command.Dir = p.Directory
	command.Env = p.Environment
	command.Stdin = p.Stdin
	command.Stdout = p.Stdout
	command.Stderr = p.Stderr
	supervisor, err := prepare(ctx, command, p.TreeID)
	if err != nil {
		return 1, err
	}
	defer supervisor.Close()
	if err := command.Start(); err != nil {
		return 1, err
	}
	if err := supervisor.Start(); err != nil {
		killErr := command.Process.Kill()
		if errors.Is(killErr, os.ErrProcessDone) {
			killErr = nil
		}
		waitErr := command.Wait()
		var exit *exec.ExitError
		if errors.As(waitErr, &exit) {
			// A killed child's exit status is expected; Wait has reaped it.
			waitErr = nil
		}
		finishErr := supervisor.Finish()
		if killErr != nil || waitErr != nil || finishErr != nil {
			return 1, errors.Join(err, ErrTreeUnconfirmed, killErr, waitErr, finishErr)
		}
		return 1, err
	}
	err = command.Wait()
	if finishErr := supervisor.Finish(); finishErr != nil {
		if finishErr == errTreeInterrupted {
			var exit *exec.ExitError
			if err != nil && !errors.As(err, &exit) {
				return 1, err
			}
			code := processExitCode(command.ProcessState)
			if code == 0 {
				code = 1
			}
			return code, nil
		}
		return 1, errors.Join(ErrTreeUnconfirmed, finishErr)
	}
	if err == nil {
		if ctx.Err() != nil {
			return 1, nil
		}
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return processExitCode(exit.ProcessState), nil
	}
	return 1, err
}
