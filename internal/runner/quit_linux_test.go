package runner

import (
	"syscall"
	"testing"
)

func TestForwardQuitRetainedNode(t *testing.T) {
	testNodeSignalDeliveryWithSignal(t, Execute, false, syscall.SIGQUIT, "SIGQUIT")
}

func TestForwardGroupQuitRetainedNode(t *testing.T) {
	testNodeSignalDeliveryWithSignal(t, Execute, true, syscall.SIGQUIT, "SIGQUIT")
}
