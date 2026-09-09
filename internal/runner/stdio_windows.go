package runner

import (
	"os"

	"golang.org/x/sys/windows"
)

// nativeStdio owns only duplicated child handles. Original files stay owned by
// the caller; closing this after CreateProcess is necessary for pipe EOF.
type nativeStdio struct {
	startup windows.StartupInfo
	handles []windows.Handle
}

func duplicateNativeStdio(files [3]*os.File) (*nativeStdio, error) {
	s := &nativeStdio{}
	for _, file := range files {
		if file == nil {
			s.close()
			return nil, os.ErrInvalid
		}
		source := windows.Handle(file.Fd())
		// A closed os.File reports -1, also the Windows current-process
		// pseudo-handle. Never pass that value to DuplicateHandle as stdio.
		if source == windows.InvalidHandle {
			s.close()
			return nil, &os.PathError{Op: "duplicate stdio", Path: file.Name(), Err: os.ErrClosed}
		}
		var handle windows.Handle
		if err := windows.DuplicateHandle(windows.CurrentProcess(), source, windows.CurrentProcess(), &handle, 0, true, windows.DUPLICATE_SAME_ACCESS); err != nil {
			s.close()
			return nil, err
		}
		s.handles = append(s.handles, handle)
	}
	s.startup = windows.StartupInfo{Flags: windows.STARTF_USESTDHANDLES, StdInput: s.handles[0], StdOutput: s.handles[1], StdErr: s.handles[2]}
	return s, nil
}

func (s *nativeStdio) close() {
	for _, handle := range s.handles {
		windows.CloseHandle(handle)
	}
	s.handles = nil
}
