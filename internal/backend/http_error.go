package backend

import (
	"context"
	"errors"
	"net"
)

// HTTP errors may contain credential-bearing URLs, including in nested proxy
// errors. Keep the cause for classification without formatting its text.
type requestFailure struct {
	cause      error
	invalidURL bool
}

func (e *requestFailure) Error() string {
	if e.invalidURL {
		return "DOWNLOAD_FAILED: invalid download request; check the configured source URL"
	}
	if errors.Is(e.cause, context.Canceled) {
		return "DOWNLOAD_FAILED: request canceled"
	}
	var timed net.Error
	if errors.Is(e.cause, context.DeadlineExceeded) || errors.As(e.cause, &timed) && timed.Timeout() {
		return "DOWNLOAD_FAILED: request timed out"
	}
	return "DOWNLOAD_FAILED: request failed; check network, proxy and TLS settings"
}

func (e *requestFailure) Unwrap() error { return e.cause }
