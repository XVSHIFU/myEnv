package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWindowsConsoleInterruptWaitsForNode(t *testing.T) {
	testWindowsConsoleSignal(t, "SIGINT", 0)
}

func TestWindowsConsoleBreakWaitsForNode(t *testing.T) {
	testWindowsConsoleSignal(t, "SIGBREAK", 1)
}

func TestWindowsConsoleDefaultInterruptExit(t *testing.T) {
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
	run := func(managed bool) int {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		marker := filepath.Join(t.TempDir(), "ready")
		args := []string{"-e", `require('fs').writeFileSync(process.argv[1],'ready');setInterval(()=>{},1000)`, marker}
		if !managed {
			// Protect only the baseline test driver. The managed branch must
			// obtain its protection from the production runner itself.
			observed := make(chan os.Signal, 1)
			signal.Notify(observed, os.Interrupt)
			defer signal.Stop(observed)
		}
		sent := make(chan error, 1)
		go func() {
			for {
				if _, err := os.Stat(marker); err == nil {
					break
				}
				if ctx.Err() != nil {
					sent <- ctx.Err()
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			ok, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("GenerateConsoleCtrlEvent").Call(0, 0)
			if ok == 0 {
				sent <- err
			} else {
				sent <- nil
			}
		}()
		var code int
		var runErr error
		if managed {
			code, runErr = Execute(ctx, Process{Executable: prepared.Executable, Args: args, Environment: os.Environ()})
		} else {
			command := exec.CommandContext(ctx, prepared.Executable, args...)
			runErr = command.Run()
			var exited *exec.ExitError
			if errors.As(runErr, &exited) {
				code = exited.ExitCode()
				runErr = nil
			}
		}
		eventErr := <-sent
		if runErr != nil || eventErr != nil || ctx.Err() != nil || code == 0 {
			t.Fatalf("managed=%v code=%d run=%v event=%v context=%v", managed, code, runErr, eventErr, ctx.Err())
		}
		return code
	}
	baseline := run(false)
	managed := run(true)
	if managed != baseline {
		t.Fatalf("native interrupt exit changed: direct=%d managed=%d", baseline, managed)
	}
	t.Logf("direct and managed Ctrl-C exit=%d", managed)
}

func privateRunnerConsole(t *testing.T) bool {
	t.Helper()
	if os.Getenv("MYENV_PRIVATE_RUN_CONSOLE") != t.Name() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
		cmd.Env = append(os.Environ(), "MYENV_PRIVATE_RUN_CONSOLE="+t.Name())
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, HideWindow: true}
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("private console: %v: %s", err, output)
		}
		t.Logf("%s", output)
		return false
	}
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	var ids [8]uint32
	count, _, err := kernel.NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids)))
	if count != 1 || ids[0] != uint32(os.Getpid()) {
		t.Fatalf("console not private: %d %v %v", count, ids, err)
	}
	if ok, _, err := kernel.NewProc("SetConsoleCtrlHandler").Call(0, 0); ok == 0 {
		t.Fatal(err)
	}
	return true
}

func testWindowsConsoleSignal(t *testing.T, signalName string, event uintptr) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	if !privateRunnerConsole(t) {
		return
	}
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "signal")
	child := `const fs=require('fs');process.on('SIGINT',()=>{fs.writeFileSync(process.argv[1]+'.child','done');process.exit(0)});fs.writeFileSync(process.argv[1]+'.ready','ready');setInterval(()=>{},1000)`
	parent := `const fs=require('fs');let signals=0;process.on('SIGINT',()=>{if(++signals!==1)process.exit(89);fs.writeFileSync(process.argv[1]+'.parent','done');setTimeout(()=>process.exit(23),150)});require('child_process').spawn(process.execPath,['-e',process.argv[2],process.argv[1]],{stdio:'inherit'});setInterval(()=>{},1000)`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	parent = strings.ReplaceAll(parent, "SIGINT", signalName)
	child = strings.ReplaceAll(child, "SIGINT", signalName)
	defer cancel()
	type result struct {
		code int
		err  error
	}
	done := make(chan result, 1)
	go func() {
		code, err := Execute(ctx, Process{Executable: prepared.Executable, Args: []string{"-e", parent, marker, child}, Environment: os.Environ()})
		done <- result{code, err}
	}()
	joined := false
	defer func() {
		cancel()
		if !joined {
			<-done
		}
	}()
	for {
		if _, err := os.Stat(marker + ".ready"); err == nil {
			break
		}
		if ctx.Err() != nil {
			t.Fatal("Node child not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ok, _, err := kernel.NewProc("GenerateConsoleCtrlEvent").Call(event, 0); ok == 0 {
		t.Fatal(err)
	}
	got := <-done
	joined = true
	if got.err != nil || got.code != 23 {
		t.Fatalf("interrupt exit: %d %v", got.code, got.err)
	}
	for _, suffix := range []string{".parent", ".child"} {
		data, err := os.ReadFile(marker + suffix)
		if err != nil || string(data) != "done" {
			t.Fatalf("missing signal handling %s: %q %v", suffix, data, err)
		}
	}
}

func TestWindowsConsoleInterruptBeforeStart(t *testing.T) {
	if !privateRunnerConsole(t) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	marker := filepath.Join(t.TempDir(), "executed")
	executable, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Fatal(err)
	}
	code, runErr := executeDirectWithPreparation(ctx, Process{Executable: executable, Args: []string{"/d", "/c", "echo executed>executed"}, Directory: filepath.Dir(marker)}, func(ctx context.Context, command *exec.Cmd, treeID string) (*supervision, error) {
		supervisor, err := prepareSupervision(ctx, command, treeID)
		if err != nil {
			return nil, err
		}
		observed := make(chan os.Signal, 1)
		signal.Notify(observed, os.Interrupt)
		if ok, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("GenerateConsoleCtrlEvent").Call(0, 0); ok == 0 {
			signal.Stop(observed)
			supervisor.Close()
			return nil, err
		}
		select {
		case <-observed:
		case <-ctx.Done():
			signal.Stop(observed)
			supervisor.Close()
			return nil, ctx.Err()
		}
		// Stop synchronizes with signal delivery; the production registration
		// remains active and has received the same real console event.
		signal.Stop(observed)
		return supervisor, nil
	})
	if code != 1 || !errors.Is(runErr, context.Canceled) || errors.Is(runErr, ErrTreeUnconfirmed) || ctx.Err() != nil {
		t.Fatalf("queued interrupt: %d %v context=%v", code, runErr, ctx.Err())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("canceled command executed: %v", err)
	}
}
