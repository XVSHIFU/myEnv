package runner

import (
	"context"
	"errors"
	"fmt"
)

func executePlatform(ctx context.Context, p Process) (int, error) {
	code, complete, err := executeLinuxSupervisor(ctx, p)
	if !complete {
		return code, fmt.Errorf("%w: %v", ErrTreeUnconfirmed, err)
	}
	// Preserve run's native exit-code contract after a started tree is canceled.
	// Backend callers recover ctx.Err through their operation wrapper.
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return code, nil
	}
	return code, err
}
