package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsNativeCancelAfterCreation(t *testing.T) {
	testWindowsNativeCancelAfterCreation(t, false)
}

func TestWindowsNativeInterruptAfterCreation(t *testing.T) {
	if !privateRunnerConsole(t) {
		return
	}
	testWindowsNativeCancelAfterCreation(t, true)
}

func testWindowsNativeCancelAfterCreation(t *testing.T, interrupt bool) {
	command, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	deadline, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	ctx, cancel := context.WithCancel(deadline)
	defer cancel()
	created := false
	var observed windows.Handle
	defer func() {
		if observed != 0 {
			windows.CloseHandle(observed)
		}
	}()
	code, err := executeWindowsWithCreation(ctx, Process{Executable: command, Args: []string{"/d", "/c", "echo executed>marker"}, Directory: root}, func(p Process, job windows.Handle, stdio windows.StartupInfo, handles []windows.Handle) (windows.ProcessInformation, error) {
		process, err := createProcessInJob(p, job, stdio, handles)
		if err != nil {
			return process, err
		}
		created = true
		// Keep an independent stable observation handle across launcher cleanup.
		if err := windows.DuplicateHandle(windows.CurrentProcess(), process.Process, windows.CurrentProcess(), &observed, windows.SYNCHRONIZE, false, 0); err != nil {
			windows.TerminateProcess(process.Process, 1)
			windows.CloseHandle(process.Process)
			windows.CloseHandle(process.Thread)
			return windows.ProcessInformation{}, err
		}
		if interrupt {
			observedSignal := make(chan os.Signal, 1)
			signal.Notify(observedSignal, os.Interrupt)
			var signalErr error
			if ok, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("GenerateConsoleCtrlEvent").Call(0, 0); ok == 0 {
				signalErr = err
			} else {
				select {
				case <-observedSignal:
				case <-ctx.Done():
					signalErr = ctx.Err()
				}
			}
			// Synchronize delivery without removing the launcher's registration.
			signal.Stop(observedSignal)
			if signalErr != nil {
				windows.TerminateProcess(process.Process, 1)
				windows.CloseHandle(process.Process)
				windows.CloseHandle(process.Thread)
				return windows.ProcessInformation{}, signalErr
			}
		} else {
			cancel()
		}
		return process, nil
	})
	if !created || code != 1 || !errors.Is(err, context.Canceled) || errors.Is(err, ErrTreeUnconfirmed) || deadline.Err() != nil {
		t.Fatalf("creation cancellation: created=%t code=%d err=%v fallback=%v", created, code, err, deadline.Err())
	}
	if interrupt && ctx.Err() != nil {
		t.Fatal("context cancellation masked console event")
	}
	if event, err := windows.WaitForSingleObject(observed, 0); err != nil || event != windows.WAIT_OBJECT_0 {
		t.Fatalf("created process survived: %d %v", event, err)
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); !os.IsNotExist(err) {
		t.Fatalf("command executed before cancellation: %v", err)
	}
}
