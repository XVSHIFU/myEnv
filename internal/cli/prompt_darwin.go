package cli

import (
	"bytes"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

func openPromptTerminal(file *os.File) (int, error) {
	var original unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &original); err != nil {
		return -1, err
	}
	if original.Mode&unix.S_IFMT != unix.S_IFCHR {
		return -1, unix.ENOTTY
	}
	// Darwin's fcntl F_GETPATH writes at most MAXPATHLEN bytes. The existing
	// binding accepts an integer argument; pin the heap buffer across that call.
	buffer := new([unix.PathMax]byte)
	var pinned runtime.Pinner
	pinned.Pin(buffer)
	_, err := unix.FcntlInt(file.Fd(), unix.F_GETPATH, int(uintptr(unsafe.Pointer(&buffer[0]))))
	pinned.Unpin()
	runtime.KeepAlive(buffer)
	if err != nil {
		return -1, err
	}
	end := bytes.IndexByte(buffer[:], 0)
	if end <= 0 {
		return -1, fmt.Errorf("invalid terminal device path")
	}
	path := string(buffer[:end])
	if !filepath.IsAbs(path) {
		return -1, fmt.Errorf("terminal device path is not absolute")
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOCTTY, 0)
	if err != nil {
		return -1, err
	}
	var reopened unix.Stat_t
	if err = unix.Fstat(fd, &reopened); err != nil {
		unix.Close(fd)
		return -1, err
	}
	if reopened.Mode&unix.S_IFMT != unix.S_IFCHR || reopened.Dev != original.Dev || reopened.Ino != original.Ino || reopened.Rdev != original.Rdev {
		unix.Close(fd)
		return -1, fmt.Errorf("terminal device changed while opening prompt input")
	}
	return fd, nil
}
