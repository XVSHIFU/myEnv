package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

var cancelPromptIO = windows.NewLazySystemDLL("kernel32.dll").NewProc("CancelSynchronousIo")

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
	type readResult struct {
		n   int
		err error
	}
	type threadResult struct {
		handle windows.Handle
		err    error
	}
	ready := make(chan threadResult, 1)
	done := make(chan readResult, 1)
	release := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		handle, err := windows.OpenThread(windows.THREAD_TERMINATE, false, windows.GetCurrentThreadId())
		ready <- threadResult{handle, err}
		if err != nil {
			return
		}
		n, err := r.file.Read(p)
		done <- readResult{n, err}
		// Keep this thread dedicated until the caller stops issuing cancellation.
		<-release
	}()
	thread := <-ready
	if thread.err != nil {
		return 0, thread.err
	}
	defer close(release)
	defer windows.CloseHandle(thread.handle)
	select {
	case result := <-done:
		if err := r.ctx.Err(); err != nil {
			return 0, err
		}
		return result.n, result.err
	case <-r.ctx.Done():
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var cancelErr error
	for {
		select {
		case <-done:
			return 0, errors.Join(r.ctx.Err(), cancelErr)
		default:
		}
		ok, _, err := cancelPromptIO.Call(uintptr(thread.handle))
		if ok == 0 && !errors.Is(err, windows.ERROR_NOT_FOUND) && cancelErr == nil {
			cancelErr = err
		}
		// Cancellation can race the beginning of ReadConsole. Retry until the read
		// actually completes; never return while it still owns the caller's buffer.
		select {
		case <-done:
			return 0, errors.Join(r.ctx.Err(), cancelErr)
		case <-ticker.C:
		}
	}
}
