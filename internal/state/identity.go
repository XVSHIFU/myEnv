package state

import (
	"errors"
	"os"
)

// ErrProcessGone is returned only when the OS confirms absence or termination.
// Access errors and incomplete observations are not evidence of process death.
var ErrProcessGone = errors.New("process no longer exists")

func supervisorIdentity() (string, error) { return processIdentity(os.Getpid()) }
