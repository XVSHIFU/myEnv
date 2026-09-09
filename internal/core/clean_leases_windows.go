package core

import (
	"encoding/hex"
	"myenv/internal/runner"
	"myenv/internal/state"
	"strconv"
	"strings"
)

func init() {
	leaseReclaimable = func(_ string, r state.LeaseRecord) (bool, error) { return windowsLeaseReclaimable(r) }
}

func windowsLeaseReclaimable(r state.LeaseRecord) (bool, error) {
	parts := strings.Split(r.Identity, ":")
	if len(parts) != 3 || parts[0] != "windows-v2" || len(parts[2]) != 16 {
		return false, nil
	}
	if _, err := hex.DecodeString(parts[2]); err != nil {
		return false, nil
	}
	if parts[2] != strings.ToLower(parts[2]) {
		return false, nil
	}
	session, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return false, nil
	}
	if strconv.FormatUint(session, 10) != parts[1] {
		return false, nil
	}
	if !managedGenerationID(r.ID) {
		return false, nil
	}
	exited, err := state.SupervisorExited(r.PID, r.Identity)
	if err != nil || !exited {
		return false, err
	}
	return runner.JobTreeGone(r.ID, uint32(session))
}
