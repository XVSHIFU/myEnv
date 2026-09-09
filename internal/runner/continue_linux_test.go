package runner

import (
	"syscall"
	"testing"
)

func TestForwardContinueRetainedNode(t *testing.T) {
	testNodeSignalDeliveryWithSignal(t, Execute, false, syscall.SIGCONT, "SIGCONT")
}
