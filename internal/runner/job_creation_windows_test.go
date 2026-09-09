package runner

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Native API feasibility regression; production still uses exec.Cmd followed
// by assignment. No user code is resumed during this test.
func TestWindowsJobAtProcessCreation(t *testing.T) {
	job, err := createSupervisionJob("")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if job != 0 {
			windows.CloseHandle(job)
		}
	}()
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		t.Fatal(err)
	}
	path, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Fatal(err)
	}
	process, err := createProcessInJob(Process{Executable: path, Args: []string{"/d", "/c", "exit", "0"}, Directory: t.TempDir()}, job, windows.StartupInfo{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(process.Process)
	defer windows.CloseHandle(process.Thread)
	defer windows.TerminateProcess(process.Process, 1)
	var accounting struct {
		Times                                 [4]int64
		PageFaults, Total, Active, Terminated uint32
	}
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
		t.Fatal(err)
	}
	if accounting.Active != 1 {
		t.Fatalf("created child not in Job: %+v", accounting)
	}
	if marker := os.Getenv("MYENV_NATIVE_CREATION_CRASH"); marker != "" {
		if err := os.WriteFile(marker, []byte(strconv.FormatUint(uint64(process.ProcessId), 10)), 0600); err != nil {
			t.Fatal(err)
		}
		// The parent test kills this process without running our defers.
		time.Sleep(30 * time.Second)
		t.Fatal("creation crash helper was not terminated")
	}
	if err := windows.CloseHandle(job); err != nil {
		t.Fatal(err)
	}
	job = 0
	if event, err := windows.WaitForSingleObject(process.Process, 3000); err != nil || event != windows.WAIT_OBJECT_0 {
		t.Fatalf("created child survived Job close: %d %v", event, err)
	}
}
