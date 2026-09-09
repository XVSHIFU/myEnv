package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"myenv/internal/backend"
	"myenv/internal/state"
)

// This fixture blocks the version probe; it never impersonates a successful uv.
func init() {
	if root := os.Getenv("MYENV_TEST_SYNC_RUNNING_OWNER"); root != "" && len(os.Args) == 2 && os.Args[1] == "--version" {
		if err := os.WriteFile(filepath.Join(root, "ready"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			os.Exit(91)
		}
		time.Sleep(30 * time.Second)
		os.Exit(92)
	}
}

func TestSyncKilledBeforeBackendLaunch(t *testing.T) {
	testSyncKilledDuringPreparation(t, false)
}

func TestSyncKilledDuringBackendProbe(t *testing.T) {
	testSyncKilledDuringPreparation(t, true)
}

func testSyncKilledDuringPreparation(t *testing.T, running bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ownerEnv := "MYENV_TEST_SYNC_EARLY_OWNER"
	if running {
		ownerEnv = "MYENV_TEST_SYNC_RUNNING_OWNER"
	}
	if root := os.Getenv(ownerEnv); root != "" {
		service := &Service{UV: &backend.UV{Executable: filepath.Join(root, "unused-uv.exe")}}
		if running {
			service.UV.Executable = os.Args[0]
		}
		_, err := service.Sync(ctx, SyncRequest{Directory: root, Progress: func(phase SyncPhase) {
			if phase == PhaseBackend && !running {
				if err := os.WriteFile(filepath.Join(root, "ready"), []byte("ready"), 0600); err != nil {
					t.Fatal(err)
				}
				<-ctx.Done()
			}
		}})
		t.Fatalf("owner survived barrier: %v", err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, executable, "-test.run=^"+t.Name()+"$")
	command.Env = append(os.Environ(), ownerEnv+"="+root)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	for {
		if _, err := os.Stat(filepath.Join(root, "ready")); err == nil {
			break
		}
		if ctx.Err() != nil {
			t.Fatal("owner never reached backend phase")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var backendProcess windows.Handle
	if running {
		data, err := os.ReadFile(filepath.Join(root, "ready"))
		if err != nil {
			t.Fatal(err)
		}
		pid, err := strconv.ParseUint(string(data), 10, 32)
		if err != nil {
			t.Fatal(err)
		}
		backendProcess, err = windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(backendProcess)
		if status, err := windows.WaitForSingleObject(backendProcess, 0); err != nil || status != uint32(windows.WAIT_TIMEOUT) {
			t.Fatalf("backend was not live: %d %v", status, err)
		}
	}
	store, err := state.OpenReadOnly(ctx, filepath.Join(root, ".myenv", "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	owners, err := store.PreparingChildOperations(ctx, "")
	if err != nil || len(owners) != 1 || !owners[0].Tracked {
		t.Fatalf("early registration: %+v %v", owners, err)
	}
	children, err := store.PreparationChildren(ctx, owners[0].ID, "")
	wantChildren := 0
	if running {
		wantChildren = 1
	}
	if err != nil || len(children) != wantChildren {
		t.Fatalf("unexpected launch: %+v %v", children, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	blocked, stop := context.WithTimeout(ctx, 80*time.Millisecond)
	live, err := (&Service{}).Clean(blocked, root, false, nil)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) || live.Changed {
		t.Fatalf("live clean: %+v %v", live, err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("owner was not killed")
	}
	waited = true
	if running {
		if status, err := windows.WaitForSingleObject(backendProcess, 3000); err != nil || status != windows.WAIT_OBJECT_0 {
			t.Fatalf("backend survived owner death: %d %v", status, err)
		}
	}
	preview, err := (&Service{}).Clean(ctx, root, true, nil)
	if err != nil || preview.Candidates != 1 || preview.Changed {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	actual, err := (&Service{}).Clean(ctx, root, false, nil)
	if err != nil || actual.RecoveredPreparations != 1 || actual.Removed != 1 || actual.Bytes != preview.Bytes {
		t.Fatalf("recovery: %+v %v", actual, err)
	}
	if _, err := os.Stat(filepath.Join(root, "myenv.lock")); !os.IsNotExist(err) {
		t.Fatalf("early crash changed lock: %v", err)
	}
}
