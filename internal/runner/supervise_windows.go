package runner

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

func prepareSupervision(ctx context.Context, command *exec.Cmd, treeID string) (*supervision, error) {
	job, err := createSupervisionJob(treeID)
	if err != nil {
		return nil, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		windows.CloseHandle(job)
		return nil, err
	}
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	// Console control events already reach children sharing this console.
	// Keep the caller alive to wait for their handlers and the entire Job;
	// forwarding the event again would deliver a duplicate to the child.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	command.Cancel = func() error {
		err := windows.TerminateJobObject(job, 1)
		_ = command.Process.Kill() // Covers cancellation before job assignment.
		return err
	}
	// CommandContext stops watching when its direct child exits. Keep watching
	// the job while inherited pipes or independent descendants remain alive.
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			_ = windows.TerminateJobObject(job, 1)
		case <-done:
		}
	}()
	var leader windows.Handle
	start := func() error {
		process, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_INFORMATION, false, uint32(command.Process.Pid))
		if err != nil {
			return err
		}
		defer func() {
			if leader == 0 {
				windows.CloseHandle(process)
			}
		}()
		if err = windows.AssignProcessToJobObject(job, process); err != nil {
			return fmt.Errorf("assign child to job: %w", err)
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		select {
		case <-interrupts:
			return context.Canceled
		default:
		}
		if err := resumeInitialThread(process, uint32(command.Process.Pid)); err != nil {
			return err
		}
		leader = process
		return nil
	}
	finish := func() error {
		interrupted := false
		// Bounded accounting queries keep the lease until all descendants exit.
		type accounting struct {
			TotalUser, TotalKernel, PeriodUser, PeriodKernel                 int64
			PageFaults, TotalProcesses, ActiveProcesses, TerminatedProcesses uint32
		}
		for {
			var info accounting
			if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil); err != nil {
				return err
			}
			if info.ActiveProcesses == 0 {
				if interrupted {
					return errTreeInterrupted
				}
				return nil
			}
			select {
			case <-interrupts:
				interrupted = true
			default:
			}
			if ctx.Err() != nil || interrupted {
				if err := windows.TerminateJobObject(job, 1); err != nil {
					return err
				}
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	var completion chan error
	return &supervision{Start: func() error {
		if err := start(); err != nil {
			return err
		}
		completion = make(chan error, 1)
		go func() {
			var result error
			defer func() {
				// Publish completion only after releasing the observation handle.
				windows.CloseHandle(leader)
				completion <- result
			}()
			// Cmd.Wait also waits for pipe copying. A detached descendant can
			// retain those pipes, so observe leader exit independently before
			// handling interrupts during the remaining Job lifetime.
			event, err := windows.WaitForSingleObject(leader, windows.INFINITE)
			if err != nil {
				result = err
				return
			}
			if event != windows.WAIT_OBJECT_0 {
				result = fmt.Errorf("wait for job leader: unexpected event %d", event)
				return
			}
			result = finish()
		}()
		return nil
	}, Finish: func() error {
		if completion != nil {
			return <-completion
		}
		return finish()
	}, Close: func() {
		close(done)
		<-stopped
		windows.CloseHandle(job)
		signal.Stop(interrupts)
	}}, nil
}
