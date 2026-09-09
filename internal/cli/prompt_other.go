//go:build !windows && !linux && !darwin

package cli

import (
	"context"
	"io"
)

func promptInput(_ context.Context, in io.Reader) io.Reader { return in }
