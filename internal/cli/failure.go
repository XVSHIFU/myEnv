package cli

import (
	"context"
	"errors"
	"myenv/internal/config"
	"myenv/internal/runner"
	"os"
	"strings"
)

func describeFailure(err error, operationError bool) (*failure, int) {
	f := &failure{Code: "USAGE_ERROR", Message: err.Error(), NextAction: "Run myenv --help."}
	exit := 2
	if operationError {
		f.Code = "INVALID_CONFIG"
		f.NextAction = "Check project input files and retry."
	}
	var pathErr *os.PathError
	var executionErr *runFailure
	if errors.As(err, &executionErr) {
		f.Code = "RUN_FAILED"
		f.NextAction = executionErr.nextAction()
		exit = 1
	}
	for _, code := range []string{"INPUT_CHANGED", "LOCK_OUT_OF_DATE", "DOWNLOAD_FAILED", "CHECKSUM_MISMATCH", "SYNC_FAILED", "ENV_NOT_READY"} {
		if strings.HasPrefix(err.Error(), code+":") {
			f.Code = code
			f.NextAction = "Check the reported cause and retry myenv sync; the prior applied generation is retained."
			exit = 1
		}
	}
	if errors.As(err, &pathErr) && f.Code != "ENV_NOT_READY" {
		f.Code = "IO_ERROR"
		f.NextAction = "Check that the project directory exists and is accessible, then retry."
		exit = 1
	}
	var needs *config.NeedsInput
	var environmentErr *environmentConfigError
	if errors.As(err, &environmentErr) {
		f.NextAction = environmentErr.nextAction()
	}
	if errors.As(err, &needs) {
		f.Code = "NEEDS_INPUT"
		f.NextAction = needs.Message
		exit = 3
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		f.Code = "CANCELED"
		f.NextAction = "Inspect current status before retrying; completed changes are retained."
		exit = 130
	}
	if executionErr != nil && errors.Is(err, runner.ErrTreeUnconfirmed) {
		f.NextAction = executionErr.nextAction()
	}
	return f, exit
}
