//go:build windows

package backend

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"time"
)

// Newly extracted SDK directories can be temporarily held without delete sharing.
// Retry only sharing/access errors, for a bounded period and without changing ACLs.
func renameSDK(ctx context.Context, source, destination string) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := os.Rename(source, destination)
		if err == nil || (!errors.Is(err, windows.ERROR_ACCESS_DENIED) && !errors.Is(err, windows.ERROR_SHARING_VIOLATION)) || !time.Now().Before(deadline) {
			return err
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
