package runner

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"myenv/internal/runtrace"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// reapDescendants is only for a dedicated supervisor process with no unrelated
// children. Subreaping is process-wide; never call this in the shared CLI/test
// process or concurrently with exec.Cmd.Wait for the same children.
// The launcher must use direct file descriptors, not exec.Cmd pipe goroutines.
func reapDescendants(start func() (int, error)) (unix.WaitStatus, error) {
	status, _, err := reapDescendantsContext(context.Background(), start)
	return status, err
}

func reapDescendantsContext(ctx context.Context, start func() (int, error)) (unix.WaitStatus, bool, error) {
	return reapDescendantsSignals(ctx, start, nil)
}

func reapDescendantsSignals(ctx context.Context, start func() (int, error), forward <-chan os.Signal) (unix.WaitStatus, bool, error) {
	return reapDescendantsJobControl(ctx, start, forward, nil, nil)
}

func reapDescendantsJobControl(ctx context.Context, start func() (int, error), forward <-chan os.Signal, stopped func() error, continuing func() error) (unix.WaitStatus, bool, error) {
	if err := ctx.Err(); err != nil {
		return 0, true, err
	}
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		return 0, true, fmt.Errorf("enable descendant reaping: %w", err)
	}
	wakeup := make(chan os.Signal, 1)
	signal.Notify(wakeup, syscall.SIGCHLD)
	defer signal.Stop(wakeup)
	leader, err := start()
	if err != nil {
		return 0, true, err
	}
	var leaderStatus unix.WaitStatus
	leaderReaped := false
	// A terminal/protocol callback failure must not abandon a live tree.
	// Preserve the failure while using the cancellation drain to reach ECHILD.
	var controlErr error
	for {
		if ctx.Err() != nil || controlErr != nil {
			if err := signalOwnedChildren(unix.SIGKILL); err != nil {
				return leaderStatus, false, errors.Join(controlErr, err)
			}
		}
		var status unix.WaitStatus
		pid, err := unix.Wait4(-1, &status, unix.WNOHANG|unix.WUNTRACED|unix.WCONTINUED, nil)
		if err == unix.EINTR {
			continue
		}
		if err == unix.ECHILD {
			if !leaderReaped {
				return 0, false, fmt.Errorf("leader termination was not observed")
			}
			if controlErr != nil {
				return leaderStatus, true, errors.Join(controlErr, ctx.Err())
			}
			return leaderStatus, true, ctx.Err()
		}
		if err != nil {
			return 0, false, fmt.Errorf("wait for descendants: %w", err)
		}
		// Stop/continue events are observable state changes, not reaping.
		// In particular, never overwrite the leader's final exit status or
		// allow a stopped leader to satisfy the completion condition.
		if pid > 0 && (status.Stopped() || status.Continued()) {
			if !leaderReaped && pid == leader && status.Stopped() && stopped != nil && ctx.Err() == nil && controlErr == nil {
				if err := stopped(); err != nil {
					controlErr = err
				}
			}
			continue
		}
		if !leaderReaped && pid == leader {
			leaderStatus, leaderReaped = status, true
			runtrace.Mark("leader_reaped")
		}
		if pid == 0 {
			if ctx.Err() == nil && controlErr == nil {
				select {
				case <-wakeup:
				case <-ctx.Done():
				case sig := <-forward:
					// Once reaped, the leader PID is no longer an owned handle.
					// Continue the remaining tree without targeting its old group.
					if sig == syscall.SIGCONT && continuing != nil && !leaderReaped {
						if err := continuing(); err != nil {
							controlErr = err
							continue
						}
					}
					group := 0
					// Terminal-launched leaders own a separate process group. Keep
					// stopped grandchildren (e.g. sh -> sleep) in the resume broadcast.
					// The unreaped leader pins this group identity; never reuse it later.
					if continuing != nil && !leaderReaped {
						group = leader
						if err := unix.Kill(-group, unix.Signal(sig.(syscall.Signal))); err != nil && err != unix.ESRCH {
							return leaderStatus, false, err
						}
					}
					if err := signalOwnedChildrenExceptGroup(unix.Signal(sig.(syscall.Signal)), group); err != nil {
						return leaderStatus, false, err
					}
				}
			} else {
				// proc children enumeration can omit racing exits. Retry while
				// waiting; only ECHILD is proof that the owned tree has ended.
				select {
				case <-wakeup:
				case <-time.After(25 * time.Millisecond):
				}
			}
		}
	}
}

// Only the dedicated reaper calls wait, on this same goroutine. Listed direct
// children cannot have their PIDs reused before we reap them. Never use this
// routine in a process with unrelated children or concurrent waiters.
func signalOwnedChildren(sig unix.Signal) error {
	return signalOwnedChildrenExceptGroup(sig, 0)
}
func signalOwnedChildrenExceptGroup(sig unix.Signal, broadcastGroup int) error {
	tasks, err := os.Open("/proc/self/task")
	if err != nil {
		return err
	}
	entries, readErr := tasks.ReadDir(4097)
	_ = tasks.Close()
	if readErr != nil && readErr != io.EOF {
		return readErr
	}
	if len(entries) > 4096 {
		return fmt.Errorf("supervisor thread limit exceeded")
	}
	for _, task := range entries {
		file, err := os.Open("/proc/self/task/" + task.Name() + "/children")
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
		_ = file.Close()
		if err != nil {
			return err
		}
		if len(data) > 1<<20 {
			return fmt.Errorf("supervisor child list limit exceeded")
		}
		for _, value := range strings.Fields(string(data)) {
			pid, err := strconv.Atoi(value)
			if err != nil || pid <= 1 {
				return fmt.Errorf("invalid owned child PID")
			}
			if broadcastGroup != 0 {
				group, err := unix.Getpgid(pid)
				if err == unix.ESRCH {
					continue
				}
				if err != nil {
					return err
				}
				if group == broadcastGroup {
					continue
				}
			}
			if err := unix.Kill(pid, sig); err != nil && err != unix.ESRCH {
				return err
			}
		}
	}
	return nil
}
