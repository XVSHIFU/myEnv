package runner

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

var createNamedJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("CreateJobObjectW")
var openNamedJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenJobObjectW")

// JobTreeGone may only be used after the matching supervisor is confirmed dead.
// A missing Local name is evidence only in the original Windows session.
// Existing objects remain protective, even if their active count is zero.
func JobTreeGone(treeID string, session uint32) (bool, error) {
	var current uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &current); err != nil {
		return false, err
	}
	if current != session {
		return false, fmt.Errorf("cannot inspect job in a different Windows session")
	}
	name, err := supervisionJobName(treeID)
	if err != nil {
		return false, err
	}
	handle, _, callErr := openNamedJob.Call(4, 0, uintptr(unsafe.Pointer(name))) // JOB_OBJECT_QUERY
	if handle != 0 {
		windows.CloseHandle(windows.Handle(handle))
		return false, nil
	}
	if callErr == windows.ERROR_FILE_NOT_FOUND {
		return true, nil
	}
	return false, callErr
}

func supervisionJobName(treeID string) (*uint16, error) {
	if len(treeID) != 32 {
		return nil, fmt.Errorf("invalid supervision identity")
	}
	for _, c := range treeID {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return nil, fmt.Errorf("invalid supervision identity")
		}
	}
	return windows.UTF16PtrFromString(`Local\myEnv-lease-` + treeID)
}

func createSupervisionJob(treeID string) (windows.Handle, error) {
	if treeID == "" {
		return windows.CreateJobObject(nil, nil)
	}
	name, err := supervisionJobName(treeID)
	if err != nil {
		return 0, err
	}
	// x/sys CreateJobObject drops last-error on a successful handle. Retain
	// it here so an existing object can never be adopted and reconfigured.
	r, _, callErr := createNamedJob.Call(0, uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, callErr
	}
	handle := windows.Handle(r)
	if callErr == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(handle)
		return 0, fmt.Errorf("supervision identity already exists")
	}
	return handle, nil
}
