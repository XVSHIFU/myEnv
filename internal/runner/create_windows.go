package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// createProcessInJob returns a suspended process already assigned to job.
// The caller owns returned handles and supplies any inheritable stdio handles.
// Integration with Execute's I/O and cancellation lifecycle is separate.
func createProcessInJob(p Process, job windows.Handle, stdio windows.StartupInfo, handles []windows.Handle) (windows.ProcessInformation, error) {
	var process windows.ProcessInformation
	if !filepath.IsAbs(p.Executable) {
		return process, fmt.Errorf("executable must be absolute")
	}
	application, err := windows.UTF16PtrFromString(p.Executable)
	if err != nil {
		return process, err
	}
	commandLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{p.Executable}, p.Args...)))
	if err != nil {
		return process, err
	}
	var directory *uint16
	if p.Directory != "" {
		directory, err = windows.UTF16PtrFromString(p.Directory)
		if err != nil {
			return process, err
		}
	}
	env := p.Environment
	if env == nil {
		env = os.Environ()
	}
	for _, entry := range env {
		if strings.ContainsRune(entry, 0) {
			return process, fmt.Errorf("environment contains NUL")
		}
	}
	env, err = Environment(env, nil, nil, true)
	if err != nil {
		return process, err
	}
	hasSystemRoot := false
	for _, entry := range env {
		if strings.HasPrefix(strings.ToUpper(entry), "SYSTEMROOT=") {
			hasSystemRoot = true
		}
	}
	if !hasSystemRoot {
		env = append(env, "SystemRoot="+os.Getenv("SystemRoot"))
	}
	block := utf16.Encode([]rune(strings.Join(env, "\x00") + "\x00\x00"))
	attributes, err := windows.NewProcThreadAttributeList(2)
	if err != nil {
		return process, err
	}
	defer attributes.Delete()
	// PROC_THREAD_ATTRIBUTE_JOB_LIST from Microsoft Windows metadata.
	if err := attributes.Update(131085, unsafe.Pointer(&job), unsafe.Sizeof(job)); err != nil {
		return process, err
	}
	if len(handles) != 0 {
		if err := attributes.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
			return process, err
		}
	}
	startup := windows.StartupInfoEx{StartupInfo: stdio}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.ProcThreadAttributeList = attributes.List()
	flags := uint32(windows.CREATE_SUSPENDED | windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT)
	if p.Background {
		flags |= windows.CREATE_NO_WINDOW
	}
	err = windows.CreateProcess(application, commandLine, nil, nil, len(handles) != 0, flags, &block[0], directory, &startup.StartupInfo, &process)
	if err != nil {
		return process, &os.PathError{Op: "create process", Path: p.Executable, Err: err}
	}
	return process, err
}
