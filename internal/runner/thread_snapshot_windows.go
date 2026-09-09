package runner

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var snapshotDLL = windows.NewLazySystemDLL("kernel32.dll")
var captureThreadSnapshot = snapshotDLL.NewProc("PssCaptureSnapshot")
var freeThreadSnapshot = snapshotDLL.NewProc("PssFreeSnapshot")
var createThreadMarker = snapshotDLL.NewProc("PssWalkMarkerCreate")
var freeThreadMarker = snapshotDLL.NewProc("PssWalkMarkerFree")
var walkThreadSnapshot = snapshotDLL.NewProc("PssWalkSnapshot")

// PSS_THREAD_ENTRY from processsnapshot.h. Windows pointer and FILETIME
// alignment must be preserved; the amd64 layout is 120 bytes.
type snapshotThreadEntry struct {
	ExitStatus                                 uint32
	TebBaseAddress                             uintptr
	ProcessID, ThreadID                        uint32
	AffinityMask                               uintptr
	Priority, BasePriority                     int32
	LastSyscallFirstArgument                   uintptr
	LastSyscallNumber                          uint16
	CreateTime, ExitTime, KernelTime, UserTime windows.Filetime
	Win32StartAddress                          uintptr
	CaptureTime                                windows.Filetime
	Flags                                      uint32
	SuspendCount, SizeOfContextRecord          uint16
	ContextRecord                              uintptr
}

// The process was created suspended and assigned to its Job before this call.
// Capture only that process's thread IDs, never the system-wide thread list or
// memory pages/contexts. It must still have exactly one initial thread.
func resumeInitialThread(process windows.Handle, pid uint32) (resultErr error) {
	for _, proc := range []*windows.LazyProc{captureThreadSnapshot, freeThreadSnapshot, createThreadMarker, freeThreadMarker, walkThreadSnapshot} {
		if err := proc.Find(); err != nil {
			return err
		}
	}
	var snapshot, marker windows.Handle
	code, _, _ := captureThreadSnapshot.Call(uintptr(process), 0x80, 0, uintptr(unsafe.Pointer(&snapshot))) // PSS_CAPTURE_THREADS
	if code != 0 {
		return fmt.Errorf("capture child threads: %w", syscall.Errno(code))
	}
	defer func() {
		code, _, _ := freeThreadSnapshot.Call(uintptr(windows.CurrentProcess()), uintptr(snapshot))
		if code != 0 {
			resultErr = errors.Join(resultErr, fmt.Errorf("free thread snapshot: %w", syscall.Errno(code)))
		}
	}()
	code, _, _ = createThreadMarker.Call(0, uintptr(unsafe.Pointer(&marker)))
	if code != 0 {
		return fmt.Errorf("create thread marker: %w", syscall.Errno(code))
	}
	defer func() {
		code, _, _ := freeThreadMarker.Call(uintptr(marker))
		if code != 0 {
			resultErr = errors.Join(resultErr, fmt.Errorf("free thread marker: %w", syscall.Errno(code)))
		}
	}()
	var entry snapshotThreadEntry
	code, _, _ = walkThreadSnapshot.Call(uintptr(snapshot), 3, uintptr(marker), uintptr(unsafe.Pointer(&entry)), unsafe.Sizeof(entry)) // PSS_WALK_THREADS
	if code != 0 {
		return fmt.Errorf("read initial thread: %w", syscall.Errno(code))
	}
	if entry.ProcessID != pid || entry.ThreadID == 0 {
		return fmt.Errorf("snapshot initial thread does not match child %d", pid)
	}
	threadID := entry.ThreadID
	code, _, _ = walkThreadSnapshot.Call(uintptr(snapshot), 3, uintptr(marker), uintptr(unsafe.Pointer(&entry)), unsafe.Sizeof(entry))
	if code != uintptr(windows.ERROR_NO_MORE_ITEMS) {
		return fmt.Errorf("child %d does not have one initial thread (snapshot status %d)", pid, code)
	}
	thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, threadID)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(thread)
	count, err := windows.ResumeThread(thread)
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("unexpected initial thread suspension count %d", count)
	}
	return nil
}
