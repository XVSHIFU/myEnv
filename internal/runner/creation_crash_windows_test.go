package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsNativeCreationOwnerCrash(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "created.pid")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	helper := exec.CommandContext(ctx, executable, "-test.run=^TestWindowsJobAtProcessCreation$")
	helper.Env = append(os.Environ(), "MYENV_NATIVE_CREATION_CRASH="+marker)
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	joined := false
	defer func() {
		if !joined {
			_ = helper.Process.Kill()
			_ = helper.Wait()
		}
	}()
	var pid uint64
	for {
		data, readErr := os.ReadFile(marker)
		if readErr == nil {
			pid, err = strconv.ParseUint(string(data), 10, 32)
			if err == nil {
				break
			}
		} else if !os.IsNotExist(readErr) {
			t.Fatal(readErr)
		}
		if ctx.Err() != nil {
			t.Fatal("created process not reported")
		}
		time.Sleep(10 * time.Millisecond)
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	defer windows.TerminateProcess(handle, 1)
	if event, err := windows.WaitForSingleObject(handle, 0); err != nil || event != uint32(windows.WAIT_TIMEOUT) {
		t.Fatalf("created process not alive: %d %v", event, err)
	}
	if err := helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = helper.Wait()
	joined = true
	if err == nil {
		t.Fatal("forced helper exit succeeded")
	}
	if event, err := windows.WaitForSingleObject(handle, 2000); err != nil || event != windows.WAIT_OBJECT_0 {
		t.Fatalf("suspended child survived owner death: %d %v", event, err)
	}
	if ctx.Err() != nil {
		t.Fatal("fallback cancellation was required")
	}
}
