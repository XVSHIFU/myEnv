package runner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSupervisorReadyCatchesStop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var reads, writes []*os.File
	for i := 0; i < 3; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		reads = append(reads, r)
		writes = append(writes, w)
		defer r.Close()
		defer w.Close()
	}
	command := exec.CommandContext(ctx, os.Args[0], supervisorArgument)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.ExtraFiles = []*os.File{reads[0], writes[1], reads[2]}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
	reads[0].Close()
	writes[1].Close()
	reads[2].Close()
	decoder := json.NewDecoder(reads[1])
	var reply supervisorReply
	if err := decoder.Decode(&reply); err != nil || !reply.Ready {
		t.Fatalf("ready: %+v %v", reply, err)
	}
	status, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(command.Process.Pid), "status"))
	if err != nil {
		t.Fatal(err)
	}
	caught := false
	for _, line := range strings.Split(string(status), "\n") {
		if value, ok := strings.CutPrefix(line, "SigCgt:"); ok {
			mask, err := strconv.ParseUint(strings.TrimSpace(value), 16, 64)
			if err != nil {
				t.Fatal(err)
			}
			caught = mask&(uint64(1)<<uint(syscall.SIGTSTP-1)) != 0
		}
	}
	if !caught {
		t.Fatal("Ready emitted before SIGTSTP handler installed")
	}
	if err := command.Process.Signal(syscall.SIGTSTP); err != nil {
		t.Fatal(err)
	}
	// Reject before launching any user child; the helper must remain responsive.
	if err := json.NewEncoder(writes[0]).Encode(supervisorRequest{Executable: "relative"}); err != nil {
		t.Fatal(err)
	}
	writes[0].Close()
	if err := decoder.Decode(&reply); err != nil || !reply.Complete || reply.Error == "" {
		t.Fatalf("reply: %+v %v", reply, err)
	}
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
}
