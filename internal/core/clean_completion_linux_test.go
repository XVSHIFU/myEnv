package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"myenv/internal/runner"
	"myenv/internal/state"
)

func TestCleanCompletedLinuxOrphan(t *testing.T) {
	if root := os.Getenv("MYENV_CLEAN_COMPLETION_HELPER"); root != "" {
		ctx := context.Background()
		work := filepath.Join(root, ".myenv")
		store, err := state.Open(ctx, filepath.Join(work, "state.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		var selected *RunEnvironment
		previous := ""
		for i := 1; i <= 4; i++ {
			id := fmt.Sprintf("%032x", i)
			dir := filepath.Join(work, "generations", id)
			if err = os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, "payload"), []byte("retained"), 0600); err != nil {
				t.Fatal(err)
			}
			if err = store.Publish(ctx, state.Generation{ID: id, Directory: dir, InputDigest: id, NodeExecutable: filepath.Join(dir, "node")}, previous); err != nil {
				t.Fatal(err)
			}
			if i == 1 {
				lease, err := acquireRunLease(ctx, store)
				if err != nil {
					t.Fatal(err)
				}
				selected = leasedRunEnvironment(store, lease)
				if err = prepareRunCompletion(ctx, work, store, selected); err != nil {
					t.Fatal(err)
				}
			}
			previous = id
		}
		code, runErr := runner.Execute(ctx, runner.Process{TreeID: selected.TreeID, Completion: selected.Completion, Executable: "/bin/sh", Args: []string{"-c", "printf ready > started; exec sleep 5"}, Directory: root, Environment: os.Environ()})
		if err = selected.Finish(runErr); err != nil {
			t.Fatal(err)
		}
		if runErr != nil {
			t.Fatal(runErr)
		}
		os.Exit(code)
	}
	root := t.TempDir()
	work := filepath.Join(root, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCleanCompletedLinuxOrphan$")
	command.Env = append(os.Environ(), "MYENV_CLEAN_COMPLETION_HELPER="+root)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fixture did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	store, err := state.OpenReadOnly(ctx, filepath.Join(work, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	records, err := store.LeaseRecords(ctx, "", 128)
	store.Close()
	if err != nil || len(records) != 1 || len(records[0].CompletionToken) != 64 {
		t.Fatal("missing durable binding", records, err)
	}
	record := records[0]
	if gone, err := linuxLeaseReclaimable(work, record); err != nil || gone {
		t.Fatal("live owner reclaimable", gone, err)
	}
	if err = command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err = command.Wait(); err == nil {
		t.Fatal("caller was not killed")
	}
	deadline = time.Now().Add(time.Second)
	for {
		gone, err := linuxLeaseReclaimable(work, record)
		if err == nil && gone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("completion proof did not arrive", gone, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	path := filepath.Join(work, leaseReceiptName(record.ID))
	proof, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{nil, []byte("partial"), append(append([]byte{}, proof...), 'x')} {
		if err = os.WriteFile(path, bad, 0600); err != nil {
			t.Fatal(err)
		}
		if gone, err := linuxLeaseReclaimable(work, record); err != nil || gone {
			t.Fatal("invalid proof accepted", gone, err)
		}
	}
	if err = os.WriteFile(path, proof, 0600); err != nil {
		t.Fatal(err)
	}
	otherPath := path + ".target"
	if err = os.Rename(path, otherPath); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(otherPath, path); err != nil {
		t.Fatal(err)
	}
	if gone, err := linuxLeaseReclaimable(work, record); err == nil || gone {
		t.Fatal("symlink proof accepted", gone, err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	if gone, err := linuxLeaseReclaimable(work, record); err == nil || gone {
		t.Fatal("FIFO proof accepted", gone, err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(otherPath, path); err != nil {
		t.Fatal(err)
	}
	stale := record
	stale.CompletionToken = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if gone, err := linuxLeaseReclaimable(work, stale); err != nil || gone {
		t.Fatal("wrong token accepted", gone, err)
	}
	for _, dry := range []bool{true, false} {
		result, err := (&Service{}).Clean(ctx, root, dry, nil)
		if err != nil || result.RecoverableLeases != 1 || result.Candidates != 2 {
			t.Fatalf("clean dry=%v: %+v %v", dry, result, err)
		}
		if dry {
			if _, err = os.Stat(path); err != nil {
				t.Fatal("preview removed proof", err)
			}
		} else if result.RecoveredLeases != 1 || result.Removed != 2 {
			t.Fatal("recovery incomplete", result)
		}
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("recovered receipt remains", err)
	}
	for _, i := range []int{3, 4} {
		if _, err = os.Stat(filepath.Join(work, "generations", fmt.Sprintf("%032x", i), "payload")); err != nil {
			t.Fatal("removed current/previous", err)
		}
	}
}
