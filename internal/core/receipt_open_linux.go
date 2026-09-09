package core

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// os.Root resolves links itself, including when OpenFile receives NOFOLLOW.
// Anchor the parent using Root, then let openat reject the final link atomically.
func openCompletionReceipt(root *os.Root, name string) (*os.File, error) {
	parent, err := root.OpenRoot(filepath.Dir(name))
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	directory, err := parent.Open(".")
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	fd, err := unix.Openat(int(directory.Fd()), filepath.Base(name), unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open receipt", Path: name, Err: err}
	}
	return os.NewFile(uintptr(fd), name), nil
}
