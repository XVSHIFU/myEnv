package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func executePlatform(ctx context.Context, p Process) (int, error) {
	return executeWindowsWithCreation(ctx, p, createProcessInJob)
}

type windowsProcessCreator func(Process, windows.Handle, windows.StartupInfo, []windows.Handle) (windows.ProcessInformation, error)

func executeWindowsWithCreation(ctx context.Context, p Process, create windowsProcessCreator) (int, error) {
	job, err := createSupervisionJob(p.TreeID)
	if err != nil {
		return 1, err
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return 1, err
	}
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	streams, err := prepareNativeIO(p)
	if err != nil {
		return 1, err
	}
	defer streams.close()
	stdio, err := duplicateNativeStdio(streams.files)
	if err != nil {
		return 1, err
	}
	defer stdio.close()
	if err := ctx.Err(); err != nil {
		return 1, err
	}
	process, err := create(p, job, stdio.startup, stdio.handles)
	if err != nil {
		return 1, err
	}
	defer windows.CloseHandle(process.Process)
	defer windows.CloseHandle(process.Thread)
	stdio.close()
	streams.start()
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			_ = windows.TerminateJobObject(job, 1)
		case <-done:
		}
	}()
	defer func() { close(done); <-stopped }()
	startErr := ctx.Err()
	select {
	case <-interrupts:
		startErr = context.Canceled
	default:
	}
	if startErr == nil {
		_, startErr = windows.ResumeThread(process.Thread)
	}
	if startErr != nil {
		_ = windows.TerminateJobObject(job, 1)
	}
	event, waitErr := windows.WaitForSingleObject(process.Process, windows.INFINITE)
	if waitErr == nil && event != windows.WAIT_OBJECT_0 {
		waitErr = fmt.Errorf("unexpected process wait event %d", event)
	}
	if waitErr != nil {
		_ = windows.TerminateJobObject(job, 1)
		streams.close()
		_ = streams.wait()
		return 1, errors.Join(ErrTreeUnconfirmed, waitErr)
	}
	interrupted := false
	var finishErr error
	for {
		var info struct {
			Times                                 [4]int64
			PageFaults, Total, Active, Terminated uint32
		}
		finishErr = windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil)
		if finishErr != nil || info.Active == 0 {
			break
		}
		select {
		case <-interrupts:
			interrupted = true
		default:
		}
		if ctx.Err() != nil || interrupted {
			if finishErr = windows.TerminateJobObject(job, 1); finishErr != nil {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if finishErr != nil {
		_ = windows.TerminateJobObject(job, 1)
		streams.close()
	}
	ioErr := streams.wait()
	if finishErr != nil {
		return 1, errors.Join(ErrTreeUnconfirmed, finishErr, ioErr)
	}
	if startErr != nil {
		return 1, startErr
	}
	var code uint32
	if err := windows.GetExitCodeProcess(process.Process, &code); err != nil {
		return 1, err
	}
	if code != 0 {
		return int(code), nil
	}
	if ctx.Err() != nil || interrupted {
		return 1, nil
	}
	if ioErr != nil {
		return 1, ioErr
	}
	return 0, nil
}
