//go:build windows

package state

import (
	"context"
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// LockWorkspace holds an OS lock, released by the kernel on process exit.
// The lock file persists: unlinking it would allow two independent lock identities.
func LockWorkspace(ctx context.Context, path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	overlap := &windows.Overlapped{}
	for {
		if err = ctx.Err(); err != nil {
			file.Close()
			return nil, err
		}
		err = windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlap)
		if err == nil {
			return func() error {
				unlock := windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, overlap)
				closeErr := file.Close()
				if unlock != nil {
					return unlock
				}
				return closeErr
			}, nil
		}
		if !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			file.Close()
			return nil, err
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			file.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
