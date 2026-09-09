package runner

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"myenv/internal/runtrace"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const supervisorArgument = "__myenv_linux_supervisor_v1"

type supervisorRequest struct {
	Executable      string
	Args            []string
	Directory       string
	Environment     []string
	CompletionToken string
	TerminalGroup   int
	Deadline        time.Time
	CallerWake      bool
}
type supervisorReply struct {
	Ready    bool
	Stopped  bool
	Complete bool
	Code     int
	Error    string
	Canceled bool
}

func init() {
	if len(os.Args) == 2 && os.Args[1] == supervisorArgument {
		runtrace.Mark("helper_enter")
		code := linuxSupervisorMain()
		runtrace.Mark("helper_finished")
		runtrace.Flush()
		os.Exit(code)
	}
}

func linuxSupervisorMain() int {
	input, output := os.NewFile(3, "supervisor-request"), os.NewFile(4, "supervisor-result")
	lifetime := os.NewFile(5, "supervisor-parent-lifetime")
	if input == nil || output == nil || lifetime == nil {
		return 1
	}
	syscall.CloseOnExec(3)
	syscall.CloseOnExec(4)
	syscall.CloseOnExec(5)
	defer input.Close()
	defer output.Close()
	defer lifetime.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Only the calling process owns the write end. EOF also detects SIGKILL
	// without depending on a reusable PID or a Go runtime thread's lifetime.
	go func() {
		var b [1]byte
		_, _ = lifetime.Read(b[:])
		cancel()
	}()
	forward := make(chan os.Signal, 8)
	// Ready promises the helper can receive every signal the caller forwards.
	signal.Notify(forward, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGCONT, syscall.SIGTSTP)
	defer signal.Stop(forward)
	encoder := json.NewEncoder(output)
	if err := encoder.Encode(supervisorReply{Ready: true}); err != nil {
		return 1
	}
	reply := supervisorReply{Complete: true, Code: 1}
	runtrace.Mark("helper_ready_written")
	data, err := io.ReadAll(io.LimitReader(input, (1<<20)+1))
	if err == nil && len(data) > 1<<20 {
		err = fmt.Errorf("supervisor request exceeds 1 MiB")
	}
	var request supervisorRequest
	if err == nil {
		err = json.Unmarshal(data, &request)
	}
	if err == nil && !filepath.IsAbs(request.Executable) {
		err = fmt.Errorf("supervisor executable must be absolute")
	}
	var completion *os.File
	var callerHandle, callerAck *os.File
	if err == nil && request.CallerWake {
		callerHandle, callerAck = os.NewFile(7, "caller-pidfd"), os.NewFile(8, "caller-ack")
		syscall.CloseOnExec(7)
		syscall.CloseOnExec(8)
		defer callerHandle.Close()
		defer callerAck.Close()
	}
	stopNotified := false
	if err == nil && request.CompletionToken != "" {
		err = validateCompletionToken(request.CompletionToken)
		if err == nil {
			completion = os.NewFile(6, "supervisor-completion")
			syscall.CloseOnExec(6)
			defer completion.Close()
		}
	}
	if err == nil {
		operationContext := ctx
		runtrace.Mark("helper_request_decoded")
		if !request.Deadline.IsZero() {
			var deadlineCancel context.CancelFunc
			operationContext, deadlineCancel = context.WithDeadline(ctx, request.Deadline)
			defer deadlineCancel()
		}
		command := exec.Command(request.Executable, request.Args...)
		command.Dir, command.Env = request.Directory, request.Environment
		command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
		if request.TerminalGroup != 0 {
			foreground, terminalErr := unix.IoctlGetInt(0, unix.TIOCGPGRP)
			if terminalErr != nil || foreground != request.TerminalGroup {
				reply.Error = "terminal foreground ownership changed before launch"
				if err := encoder.Encode(reply); err != nil {
					return 1
				}
				return 0
			}
			command.SysProcAttr = &syscall.SysProcAttr{Foreground: true, Ctty: 0}
		}
		var stopped, continuing func() error
		if request.TerminalGroup != 0 {
			stopped = func() error {
				if err := restoreTerminalForeground(command, request.TerminalGroup); err != nil {
					return err
				}
				stopNotified = true
				return encoder.Encode(supervisorReply{Stopped: true})
			}
			continuing = func() error { return resumeTerminalForeground(command, request.TerminalGroup) }
		}
		status, complete, runErr := reapDescendantsJobControl(operationContext, func() (int, error) {
			runtrace.Mark("child_start_begin")
			if e := command.Start(); e != nil {
				return 0, e
			}
			runtrace.Mark("child_started")
			return command.Process.Pid, nil
		}, forward, stopped, continuing)
		runtrace.Mark("descendants_drained")
		// Only a pure cancellation may be converted to a native child exit.
		// Do not hide terminal or receipt failures that coincide with it.
		reply.Canceled = runErr == context.Canceled || runErr == context.DeadlineExceeded
		if request.TerminalGroup != 0 {
			restoreErr := restoreTerminalForeground(command, request.TerminalGroup)
			runErr = errors.Join(runErr, restoreErr)
			if restoreErr != nil {
				reply.Canceled = false
			}
		}
		if command.Process != nil {
			_ = command.Process.Release()
		}
		reply.Complete, err = complete, runErr
		if status.Exited() {
			reply.Code = status.ExitStatus()
		} else if status.Signaled() {
			reply.Code = 128 + int(status.Signal())
		}
		if reply.Canceled && reply.Code == 0 {
			reply.Code = 1
		}
		if complete && completion != nil {
			runtrace.Mark("receipt_sync_begin")
			_, receiptErr := completion.WriteAt([]byte("myenv-tree-complete-v1:"+request.CompletionToken+"\n"), 0)
			if receiptErr == nil {
				receiptErr = completion.Sync()
			}
			err = errors.Join(err, receiptErr)
			runtrace.Mark("receipt_sync_end")
			if receiptErr != nil {
				reply.Canceled = false
			}
		}
	}
	if err != nil {
		reply.Error = err.Error()
	}
	if err := encoder.Encode(reply); err != nil {
		return 1
	}
	if stopNotified && callerHandle != nil {
		if err := resumeCallerUntilAcknowledged(int(callerHandle.Fd()), int(callerAck.Fd())); err != nil {
			return 1
		}
	}
	return 0
}

// The completion bit gates lease release when the supervisor exits unexpectedly.
func executeLinuxSupervisor(ctx context.Context, p Process) (int, bool, error) {
	runtrace.Mark("supervision_begin")
	if err := ctx.Err(); err != nil {
		return 1, true, err
	}
	request := supervisorRequest{Executable: p.Executable, Args: p.Args, Directory: p.Directory, Environment: p.Environment}
	request.Deadline, _ = ctx.Deadline()
	var callerHandle, ackRead, ackWrite *os.File
	if input, ok := p.Stdin.(*os.File); ok && term.IsTerminal(int(input.Fd())) {
		foreground, err := unix.IoctlGetInt(int(input.Fd()), unix.TIOCGPGRP)
		if err != nil {
			return 1, true, err
		}
		if foreground != syscall.Getpgrp() {
			return 1, true, fmt.Errorf("terminal command requires caller in foreground")
		}
		request.TerminalGroup = foreground
		pidfd, err := unix.PidfdOpen(os.Getpid(), 0)
		if err != nil {
			return 1, true, fmt.Errorf("open caller process handle: %w", err)
		}
		callerHandle = os.NewFile(uintptr(pidfd), "caller-pidfd")
		defer callerHandle.Close()
		if err := unix.PidfdSendSignal(pidfd, 0, nil, 0); err != nil {
			return 1, true, fmt.Errorf("check caller process signaling: %w", err)
		}
		ackRead, ackWrite, err = os.Pipe()
		if err != nil {
			return 1, true, err
		}
		defer ackRead.Close()
		defer ackWrite.Close()
		request.CallerWake = true
	}
	if p.Completion != nil {
		if err := validateCompletionToken(p.Completion.Token); err != nil {
			return 1, true, err
		}
		if p.Completion.File == nil {
			return 1, true, fmt.Errorf("missing completion file")
		}
		info, err := p.Completion.File.Stat()
		if err != nil {
			return 1, true, err
		}
		if !info.Mode().IsRegular() || info.Size() != 0 {
			return 1, true, fmt.Errorf("completion file must be an empty regular file")
		}
		request.CompletionToken = p.Completion.Token
	}
	data, err := json.Marshal(request)
	if err != nil {
		return 1, true, err
	}
	if len(data) > 1<<20 {
		return 1, true, fmt.Errorf("supervisor request exceeds 1 MiB")
	}
	executable, err := os.Executable()
	if err != nil {
		return 1, true, err
	}
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		return 1, true, err
	}
	defer requestRead.Close()
	defer requestWrite.Close()
	resultRead, resultWrite, err := os.Pipe()
	if err != nil {
		return 1, true, err
	}
	defer resultRead.Close()
	defer resultWrite.Close()
	lifetimeRead, lifetimeWrite, err := os.Pipe()
	if err != nil {
		return 1, true, err
	}
	defer lifetimeRead.Close()
	defer lifetimeWrite.Close()
	command := exec.Command(executable, supervisorArgument)
	command.ExtraFiles = []*os.File{requestRead, resultWrite, lifetimeRead}
	if p.Completion != nil {
		command.ExtraFiles = append(command.ExtraFiles, p.Completion.File)
	} else if request.CallerWake {
		command.ExtraFiles = append(command.ExtraFiles, nil)
	}
	if request.CallerWake {
		command.ExtraFiles = append(command.ExtraFiles, callerHandle, ackRead)
	}
	command.Stdin, command.Stdout, command.Stderr = p.Stdin, p.Stdout, p.Stderr
	// The supervisor never shares the caller's signal-broadcast group.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	forward := make(chan os.Signal, 8)
	signal.Notify(forward, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGCONT)
	if request.TerminalGroup != 0 {
		signal.Notify(forward, syscall.SIGTSTP)
	}
	defer signal.Stop(forward)
	runtrace.Mark("helper_start_begin")
	if err := command.Start(); err != nil {
		return 1, true, err
	}
	runtrace.Mark("helper_started")
	requestRead.Close()
	resultWrite.Close()
	lifetimeRead.Close()
	if ackRead != nil {
		ackRead.Close()
	}
	cancelDone, cancelStopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(cancelStopped)
		select {
		case <-ctx.Done():
			// EOF is safe before readiness as well as while the tree runs.
			_ = lifetimeWrite.Close()
		case <-cancelDone:
		}
	}()
	defer func() { close(cancelDone); <-cancelStopped }()
	written := make(chan error, 1)
	go func() { _, e := requestWrite.Write(data); requestWrite.Close(); written <- e }()
	decoder := bufio.NewReaderSize(resultRead, (64<<10)+1)
	var ready, reply supervisorReply
	err = decodeSupervisorReply(decoder, &ready)
	runtrace.Mark("helper_ready_received")
	if err == nil && !validSupervisorReadiness(ready) {
		err = fmt.Errorf("supervisor readiness missing or invalid")
	}
	done, stopped := make(chan struct{}), make(chan struct{})
	if err == nil {
		go func() {
			defer close(stopped)
			for {
				select {
				case sig := <-forward:
					_ = command.Process.Signal(sig)
				case <-done:
					return
				}
			}
		}()
		reply, err = awaitSupervisorCompletion(ctx, decoder, request.TerminalGroup != 0, func() error {
			return syscall.Kill(os.Getpid(), syscall.SIGSTOP)
		})
		runtrace.Mark("completion_received")
		close(done)
		<-stopped
	}
	if err != nil {
		_ = lifetimeWrite.Close()
	}
	if ackWrite != nil {
		ackWrite.Close()
	}
	waitErr := command.Wait()
	runtrace.Mark("helper_waited")
	writeErr := <-written
	if err != nil {
		return 1, false, err
	}
	if waitErr != nil {
		return 1, false, waitErr
	}
	if writeErr != nil {
		return 1, reply.Complete, writeErr
	}
	if !reply.Complete {
		return 1, false, fmt.Errorf("supervisor could not confirm tree completion: %s", reply.Error)
	}
	if reply.Canceled {
		return reply.Code, true, context.Canceled
	}
	if reply.Error != "" {
		return 1, true, errors.New(reply.Error)
	}
	return reply.Code, true, nil
}

func validateCompletionToken(token string) error {
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != token {
		return fmt.Errorf("completion token must be 32 bytes of lowercase hex")
	}
	return nil
}
