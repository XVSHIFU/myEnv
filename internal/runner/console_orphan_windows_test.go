package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsConsoleInterruptAfterLeaderExit(t *testing.T) {
	testWindowsConsoleOrphan(t, false, 0)
}

func TestWindowsConsoleInterruptWithInheritedPipe(t *testing.T) {
	testWindowsConsoleOrphan(t, true, 0)
}

func TestWindowsConsoleOrphanPreservesLeaderFailure(t *testing.T) {
	testWindowsConsoleOrphan(t, true, 17)
}

func testWindowsConsoleOrphan(t *testing.T, inheritedPipe bool, leaderExit int) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	if !privateRunnerConsole(t) {
		return
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "tree")
	child := `require('fs').writeFileSync(process.argv[1]+'.ready','ready');setTimeout(()=>{},30000)`
	stdio := "ignore"
	if inheritedPipe {
		stdio = "inherit"
	}
	parent := `const fs=require('fs');const c=require('child_process').spawn(process.execPath,['-e',process.argv[2],process.argv[1]],{stdio:process.argv[3],detached:true});c.unref();const timer=setInterval(()=>{if(fs.existsSync(process.argv[1]+'.ready')){fs.writeFileSync(process.argv[1]+'.pids',JSON.stringify([process.pid,c.pid]));clearInterval(timer);setInterval(()=>{if(fs.existsSync(process.argv[1]+'.exit'))process.exit(Number(process.argv[4]))},10)}},10)`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type result struct {
		code int
		err  error
	}
	done := make(chan result, 1)
	go func() {
		var output bytes.Buffer
		code, err := Execute(ctx, Process{Executable: prepared.Executable, Args: []string{"-e", parent, marker, child, stdio, strconv.Itoa(leaderExit)}, Environment: os.Environ(), Stdout: &output})
		done <- result{code, err}
	}()
	joined := false
	defer func() {
		cancel()
		if !joined {
			<-done
		}
	}()
	var pids []uint32
	for {
		data, err := os.ReadFile(marker + ".pids")
		if err == nil && json.Unmarshal(data, &pids) == nil && len(pids) == 2 {
			break
		}
		if ctx.Err() != nil {
			t.Fatal("tree not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var handles []windows.Handle
	for _, pid := range pids {
		handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(handle)
		handles = append(handles, handle)
	}
	if err := os.WriteFile(marker+".exit", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if event, err := windows.WaitForSingleObject(handles[0], 1000); err != nil || event != windows.WAIT_OBJECT_0 {
		t.Fatalf("leader still alive: %d %v", event, err)
	}
	if event, err := windows.WaitForSingleObject(handles[1], 0); err != nil || event != uint32(windows.WAIT_TIMEOUT) {
		t.Fatalf("descendant not alive: %d %v", event, err)
	}
	if ok, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("GenerateConsoleCtrlEvent").Call(0, 0); ok == 0 {
		t.Fatal(err)
	}
	got := <-done
	joined = true
	if got.err != nil || got.code == 0 || ctx.Err() != nil {
		t.Fatalf("orphan interrupt: %+v context=%v", got, ctx.Err())
	}
	if leaderExit != 0 && got.code != leaderExit {
		t.Fatalf("leader exit lost: got %d want %d", got.code, leaderExit)
	}
	if event, err := windows.WaitForSingleObject(handles[1], 0); err != nil || event != windows.WAIT_OBJECT_0 {
		t.Fatalf("descendant survived: %d %v", event, err)
	}
}
