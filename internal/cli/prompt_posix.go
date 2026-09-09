//go:build linux || darwin

package cli

import (
	"context"
	"errors"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
	"io"
	"os"
)

type promptReader struct {
	ctx  context.Context
	file *os.File
}

func promptInput(ctx context.Context, in io.Reader) io.Reader {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return &promptReader{ctx, f}
	}
	return in
}
func (r *promptReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	// Reopen instead of dup: O_NONBLOCK must not change the caller's shared
	// open-file description. O_NOCTTY prevents acquiring a controlling terminal.
	fd, err := openPromptTerminal(r.file)
	if err != nil {
		return 0, os.NewSyscallError("open prompt input", err)
	}
	defer unix.Close(fd)
	polls := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		if err := r.ctx.Err(); err != nil {
			return 0, err
		}
		_, err := unix.Poll(polls, 50)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return 0, os.NewSyscallError("poll prompt input", err)
		}
		if err := r.ctx.Err(); err != nil {
			return 0, err
		}
		if polls[0].Revents == 0 {
			continue
		}
		n, err := unix.Read(fd, p)
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return 0, os.NewSyscallError("read prompt input", err)
		}
		if n == 0 {
			return 0, io.EOF
		}
		return n, nil
	}
}
