package runner

import (
	"errors"
	"io"
	"os"
	"reflect"
	"sync"

	"golang.org/x/sys/windows"
)

type nativeIO struct {
	files            [3]*os.File
	owned, childEnds []*os.File
	pumps            []func() error
	results          chan error
}

type nativeOutput struct {
	mu     *sync.Mutex
	writer io.Writer
}

func (w nativeOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(p)
}

func prepareNativeIO(p Process) (*nativeIO, error) {
	n := &nativeIO{}
	var outputLock sync.Mutex
	for index, value := range []any{p.Stdin, p.Stdout, p.Stderr} {
		// A shared destination uses one child pipe, preserving the write order
		// across stdout/stderr as exec.Cmd does for a comparable shared Writer.
		if index == 2 && p.Stdout != nil && reflect.TypeOf(p.Stdout).Comparable() && p.Stdout == p.Stderr {
			n.files[index] = n.files[1]
			continue
		}
		if file, ok := value.(*os.File); ok {
			if file == nil {
				n.close()
				return nil, os.ErrInvalid
			}
			n.files[index] = file
			continue
		}
		if value == nil {
			flag := os.O_WRONLY
			if index == 0 {
				flag = os.O_RDONLY
			}
			file, err := os.OpenFile(os.DevNull, flag, 0)
			if err != nil {
				n.close()
				return nil, err
			}
			n.files[index] = file
			n.owned = append(n.owned, file)
			continue
		}
		read, write, err := os.Pipe()
		if err != nil {
			n.close()
			return nil, err
		}
		n.owned = append(n.owned, read, write)
		if index == 0 {
			n.files[index] = read
			n.childEnds = append(n.childEnds, read)
			source := value.(io.Reader)
			n.pumps = append(n.pumps, func() error {
				defer write.Close()
				_, err := io.Copy(write, source)
				if pe, ok := err.(*os.PathError); ok && pe.Op == "write" && pe.Path == write.Name() && (pe.Err == windows.ERROR_BROKEN_PIPE || pe.Err == windows.ERROR_NO_DATA) {
					return nil
				}
				return err
			})
		} else {
			n.files[index] = write
			n.childEnds = append(n.childEnds, write)
			destination := nativeOutput{&outputLock, value.(io.Writer)}
			n.pumps = append(n.pumps, func() error { defer read.Close(); _, err := io.Copy(destination, read); return err })
		}
	}
	return n, nil
}
func (n *nativeIO) start() {
	for _, file := range n.childEnds {
		file.Close()
	}
	n.results = make(chan error, len(n.pumps))
	for _, pump := range n.pumps {
		go func() { n.results <- pump() }()
	}
}
func (n *nativeIO) wait() error {
	var result error
	for range n.pumps {
		result = errors.Join(result, <-n.results)
	}
	return result
}
func (n *nativeIO) close() {
	for _, file := range n.owned {
		file.Close()
	}
}
