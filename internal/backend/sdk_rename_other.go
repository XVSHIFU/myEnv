//go:build !windows

package backend

import (
	"context"
	"os"
)

func renameSDK(ctx context.Context, source, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(source, destination)
}
