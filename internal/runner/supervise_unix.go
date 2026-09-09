//go:build !windows

package runner

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func prepareSupervision(ctx context.Context, command *exec.Cmd, treeID string) (*supervision, error) {
	// A terminal child must remain in the foreground process group so reads do
	// not receive SIGTTIN. Piped commands receive a dedicated cancellation group.
	grouped := true
	if input, ok := command.Stdin.(*os.File); ok && term.IsTerminal(int(input.Fd())) {
		grouped = false
	}
	if grouped {
		command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	send := func(sig syscall.Signal) error {
		if grouped {
			return syscall.Kill(-command.Process.Pid, sig)
		}
		return command.Process.Signal(sig)
	}
	command.Cancel = func() error { return send(syscall.SIGKILL) }
	// Register before exec.Start: signals received during process creation are
	// buffered until there is a child process/group to forward them to.
	signals := make(chan os.Signal, 8)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	stop := func() { signal.Stop(signals) }
	start := func() error {
		done := make(chan struct{})
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			cancellation := ctx.Done()
			for {
				select {
				case <-cancellation:
					_ = send(syscall.SIGKILL)
					cancellation = nil
				case sig := <-signals:
					_ = send(sig.(syscall.Signal))
				case <-done:
					return
				}
			}
		}()
		stop = func() { signal.Stop(signals); close(done); <-stopped }
		return nil
	}
	// Process-group signaling does not prove that detached descendants ended.
	// Linux Execute uses its dedicated subreaper; this fallback must retain
	// protection until a platform implementation can provide equivalent proof.
	return &supervision{Start: start, Finish: func() error { return ErrTreeUnconfirmed }, Close: func() { stop() }}, nil
}
