//go:build linux || darwin

package state

import (
	"context"
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func LockWorkspace(ctx context.Context, path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	for {
		if err = ctx.Err(); err != nil {
			file.Close()
			return nil, err
		}
		err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() error {
				unlock := unix.Flock(int(file.Fd()), unix.LOCK_UN)
				closeErr := file.Close()
				if unlock != nil {
					return unlock
				}
				return closeErr
			}, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
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
