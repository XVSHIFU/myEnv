package config

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// resolveWorkspacePath follows directory junctions as well as symbolic links.
func resolveWorkspacePath(path string) (string, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	h, err := windows.CreateFile(p, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", &os.PathError{Op: "resolve", Path: path, Err: err}
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, 512)
	for {
		n, err := windows.GetFinalPathNameByHandle(h, &buf[0], uint32(len(buf)), 0)
		if err != nil {
			return "", &os.PathError{Op: "resolve", Path: path, Err: err}
		}
		if n >= uint32(len(buf)) {
			if n > 32768 {
				return "", fmt.Errorf("resolved path exceeds limit")
			}
			buf = make([]uint16, n+1)
			continue
		}
		resolved := windows.UTF16ToString(buf[:n])
		if strings.HasPrefix(resolved, `\\?\UNC\`) {
			return `\\` + resolved[8:], nil
		}
		if strings.HasPrefix(resolved, `\\?\`) && len(resolved) >= 7 && resolved[5] == ':' && resolved[6] == '\\' {
			return resolved[4:], nil
		}
		return "", fmt.Errorf("unexpected final path namespace")
	}
}
