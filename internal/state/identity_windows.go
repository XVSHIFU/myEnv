package state

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
)

func processIdentity(pid int) (string, error) {
	if pid <= 0 || uint64(pid) > 0xffffffff {
		return "", fmt.Errorf("invalid process ID")
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return "", ErrProcessGone
	}
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	status, err := windows.WaitForSingleObject(handle, 0)
	if err != nil {
		return "", err
	}
	if status == windows.WAIT_OBJECT_0 {
		return "", ErrProcessGone
	}
	if status != uint32(windows.WAIT_TIMEOUT) {
		return "", fmt.Errorf("unexpected process wait status %d", status)
	}
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return "", err
	}
	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(pid), &session); err != nil {
		return "", err
	}
	// The session scopes Local Job names. Keep it in the persisted identity,
	// so a later observer never treats a different session's absent name as
	// proof that the original tree is gone.
	return fmt.Sprintf("windows-v2:%d:%08x%08x", session, created.HighDateTime, created.LowDateTime), nil
}
