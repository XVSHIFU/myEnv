package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// The single-process candidate exists only in this assessment fixture. A passing
// assessment means the split implementation meets both contracts and the direct
// subreaper/PDEATHSIG candidate is rejected, not enabled in production.
func TestSupervisorMergeAssessment(t *testing.T) {
	if root := os.Getenv("MYENV_MERGE_ASSESSMENT_ROOT"); root != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		receipt, err := os.OpenFile(filepath.Join(root, "receipt"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		defer receipt.Close()
		args := []string{"-c", "echo $$ > leader; setsid /bin/sh -c 'echo $$ > descendant; exec sleep 30' </dev/null >/dev/null 2>&1 & wait"}
		if os.Getenv("MYENV_MERGE_ASSESSMENT_MODE") == "single" {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			command := exec.Command("/bin/sh", args...)
			command.Dir = root
			command.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
			_, complete, _ := reapDescendantsContext(ctx, func() (int, error) {
				if err := command.Start(); err != nil {
					return 0, err
				}
				return command.Process.Pid, nil
			})
			if command.Process != nil {
				command.Process.Release()
			}
			if complete {
				if _, err := receipt.WriteString("myenv-tree-complete-v1:" + strings.Repeat("a", 64) + "\n"); err != nil {
					t.Fatal(err)
				}
				if err := receipt.Sync(); err != nil {
					t.Fatal(err)
				}
			}
		} else {
			_, err := Execute(ctx, Process{Executable: "/bin/sh", Args: args, Directory: root, Environment: os.Environ(), Completion: &TreeCompletion{File: receipt, Token: strings.Repeat("a", 64)}})
			if err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	for _, mode := range []string{"split", "single"} {
		for _, event := range []string{"kill", "stop"} {
			t.Run(mode+"-"+event, func(t *testing.T) {
				root := t.TempDir()
				command := exec.Command(os.Args[0], "-test.run=^TestSupervisorMergeAssessment$")
				command.Env = append(os.Environ(), "MYENV_MERGE_ASSESSMENT_ROOT="+root, "MYENV_MERGE_ASSESSMENT_MODE="+mode)
				if err := command.Start(); err != nil {
					t.Fatal(err)
				}
				waited := false
				defer func() {
					if !waited {
						command.Process.Kill()
						command.Wait()
					}
				}()
				var handles []int
				defer func() {
					for _, fd := range handles {
						unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0)
						unix.Close(fd)
					}
				}()
				for _, name := range []string{"leader", "descendant"} {
					deadline := time.Now().Add(time.Second)
					pid := 0
					for time.Now().Before(deadline) {
						data, _ := os.ReadFile(filepath.Join(root, name))
						pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
						if pid > 1 {
							break
						}
						time.Sleep(5 * time.Millisecond)
					}
					if pid <= 1 {
						t.Fatal("fixture not ready", name)
					}
					fd, err := unix.PidfdOpen(pid, 0)
					if err != nil {
						t.Fatal(err)
					}
					handles = append(handles, fd)
				}
				if event == "kill" {
					if err := command.Process.Kill(); err != nil {
						t.Fatal(err)
					}
					command.Wait()
					waited = true
				} else {
					if err := command.Process.Signal(syscall.SIGSTOP); err != nil {
						t.Fatal(err)
					}
					var status syscall.WaitStatus
					if _, err := syscall.Wait4(command.Process.Pid, &status, syscall.WUNTRACED, nil); err != nil || !status.Stopped() {
						t.Fatal("caller did not stop", status, err)
					}
					time.Sleep(2200 * time.Millisecond)
				}
				dead := func(fd int) bool {
					p := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
					n, err := unix.Poll(p, 0)
					if err != nil {
						t.Fatal(err)
					}
					return n > 0
				}
				if mode == "split" {
					deadline := time.Now().Add(time.Second)
					for time.Now().Before(deadline) {
						data, _ := os.ReadFile(filepath.Join(root, "receipt"))
						if dead(handles[0]) && dead(handles[1]) && string(data) == "myenv-tree-complete-v1:"+strings.Repeat("a", 64)+"\n" {
							break
						}
						time.Sleep(5 * time.Millisecond)
					}
					if !dead(handles[0]) || !dead(handles[1]) {
						t.Fatal("descendants survived")
					}
					data, err := os.ReadFile(filepath.Join(root, "receipt"))
					if err != nil || string(data) != "myenv-tree-complete-v1:"+strings.Repeat("a", 64)+"\n" {
						t.Fatal("completion receipt missing", string(data), err)
					}
				} else {
					if dead(handles[1]) {
						t.Fatal("assessment failed to expose surviving grandchild")
					}
					data, err := os.ReadFile(filepath.Join(root, "receipt"))
					if err != nil || len(data) != 0 {
						t.Fatal("unexpected receipt", string(data), err)
					}
					t.Log("single-process candidate rejected: descendant alive and no receipt after caller " + event)
				}
				if event == "stop" {
					if err := command.Process.Signal(syscall.SIGCONT); err != nil {
						t.Fatal(err)
					}
					if err := command.Wait(); err != nil {
						t.Fatal(err)
					}
					waited = true
				}
			})
		}
	}
}
