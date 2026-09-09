package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsJobRetainedNode(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err = json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	t.Run("wait_for_descendant", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "completed")
		diagnostic, err := os.Create(marker + ".stderr")
		if err != nil {
			t.Fatal(err)
		}
		defer diagnostic.Close()
		child := `setTimeout(()=>require('fs').writeFileSync(process.argv[1],'done'),500)`
		parent := `const c=require('child_process').spawn(process.execPath,['-e',process.argv[1],process.argv[2]],{stdio:['ignore','ignore',2],detached:true});c.unref()`
		code, err := Execute(context.Background(), Process{TreeID: fmt.Sprintf("%032x", time.Now().UnixNano()), Executable: prepared.Executable, Args: []string{"-e", parent, child, marker}, Environment: os.Environ(), Stderr: diagnostic})
		if err != nil || code != 0 {
			t.Fatalf("exit %d: %v", code, err)
		}
		if data, err := os.ReadFile(marker); err != nil || string(data) != "done" {
			log, _ := os.ReadFile(marker + ".stderr")
			t.Fatalf("released before descendant completed: %q %v; child stderr %s", data, err, log)
		}
	})
	t.Run("cancel_descendant", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "spawned")
		parent := `const c=require('child_process').spawn(process.execPath,['-e','setTimeout(()=>{},3000)'],{stdio:'inherit'});require('fs').writeFileSync(process.argv[1],String(c.pid));setTimeout(()=>{},3000)`
		ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
		defer cancel()
		var output bytes.Buffer
		started := time.Now()
		code, err := Execute(ctx, Process{Executable: prepared.Executable, Args: []string{"-e", parent, marker}, Environment: os.Environ(), Stdout: &output, Stderr: &output})
		if err != nil || code == 0 {
			t.Fatalf("exit %d: %v %s", code, err, output.String())
		}
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("descendant never started: %v", err)
		}
		if elapsed := time.Since(started); elapsed > 2*time.Second {
			t.Fatalf("descendant retained pipe after cancellation: %s", elapsed)
		}
	})
	t.Run("cancel_after_parent_exit", func(t *testing.T) {
		parent := `const c=require('child_process').spawn(process.execPath,['-e','setTimeout(()=>{},3000)'],{stdio:'inherit',detached:true});c.unref()`
		ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
		defer cancel()
		var output bytes.Buffer
		started := time.Now()
		code, err := Execute(ctx, Process{Executable: prepared.Executable, Args: []string{"-e", parent}, Environment: os.Environ(), Stdout: &output, Stderr: &output})
		if elapsed := time.Since(started); elapsed > 2*time.Second {
			t.Fatalf("cancellation stopped watching descendants after parent exit: %s", elapsed)
		}
		if err != nil || code == 0 {
			t.Fatalf("cancelled tree returned %d: %v", code, err)
		}
	})
}

func TestWindowsJobSupervisorCrash(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err = json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	if marker := os.Getenv("MYENV_TEST_JOB_CRASH_HELPER"); marker != "" {
		program := `const fs=require('fs');const c=require('child_process').spawn(process.execPath,['-e','setTimeout(()=>{},30000)'],{stdio:'ignore',detached:true});fs.writeFileSync(process.argv[1]+'.tmp',JSON.stringify([process.pid,c.pid]));fs.renameSync(process.argv[1]+'.tmp',process.argv[1]);setTimeout(()=>{},30000)`
		code, err := Execute(context.Background(), Process{Executable: prepared.Executable, Args: []string{"-e", program, marker}, Environment: os.Environ()})
		if err != nil {
			t.Fatal(err)
		}
		if code != 0 {
			t.Fatalf("helper runtime exit %d", code)
		}
		return
	}
	marker := filepath.Join(t.TempDir(), "pids.json")
	testExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	helper := exec.CommandContext(ctx, testExecutable, "-test.run=^TestWindowsJobSupervisorCrash$")
	helper.Env = append(os.Environ(), "MYENV_TEST_JOB_CRASH_HELPER="+marker)
	if err = helper.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = helper.Process.Kill(); _ = helper.Wait() }()
	var pids []uint32
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err = os.ReadFile(marker)
		if err == nil {
			if err = json.Unmarshal(data, &pids); err != nil {
				t.Fatal(err)
			}
			break
		}
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(pids) != 2 {
		t.Fatal("supervisor did not start both Node processes")
	}
	var processes []windows.Handle
	for _, pid := range pids {
		handle, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, pid)
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(handle)
		defer windows.TerminateProcess(handle, 1) // Test-owned fallback on assertion failure.
		if event, err := windows.WaitForSingleObject(handle, 0); err != nil || event != uint32(windows.WAIT_TIMEOUT) {
			t.Fatalf("Node %d not alive: %d %v", pid, event, err)
		}
		processes = append(processes, handle)
	}
	if err = helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err = helper.Wait(); err == nil {
		t.Fatal("forced supervisor exit unexpectedly succeeded")
	}
	for i, handle := range processes {
		if event, err := windows.WaitForSingleObject(handle, 2000); err != nil || event != windows.WAIT_OBJECT_0 {
			t.Fatalf("Node %d survived supervisor: %d %v", pids[i], event, err)
		}
	}
}
