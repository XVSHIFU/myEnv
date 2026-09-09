//go:build !linux && !windows

package runner

import "context"

func executePlatform(ctx context.Context, p Process) (int, error) { return executeDirect(ctx, p) }
