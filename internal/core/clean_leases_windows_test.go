package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"myenv/internal/runner"
	"myenv/internal/state"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanKilledWindowsSupervisor(t *testing.T) {
	if os.Getenv("MYENV_LEASE_TREE_WORKER") == "1" {
		fmt.Println("tree-ready")
		for {
			time.Sleep(time.Second)
		}
	}
	if database := os.Getenv("MYENV_LEASE_TREE_DATABASE"); database != "" {
		s, err := state.Open(context.Background(), database)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		lease, err := s.AcquireActive(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(lease.ID)
		_, err = runner.Execute(context.Background(), runner.Process{TreeID: lease.ID, Executable: os.Args[0], Args: []string{"-test.run=^TestCleanKilledWindowsSupervisor$"}, Environment: append(os.Environ(), "MYENV_LEASE_TREE_WORKER=1"), Stdout: os.Stdout, Stderr: os.Stderr})
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "myenv.yaml"), []byte("schema: 1\ntools: {python: '3.12'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(project, ".myenv")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(work, "state.db")
	s, err := state.Open(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	previous := ""
	publish := func(i int) {
		id := fmt.Sprintf("%032x", i)
		dir := filepath.Join(work, "generations", id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := s.Publish(ctx, state.Generation{ID: id, Directory: dir, InputDigest: id, PythonExecutable: filepath.Join(dir, "python")}, previous); err != nil {
			t.Fatal(err)
		}
		previous = id
	}
	publish(1)
	parentLease, err := s.AcquireActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer s.ReleaseLease(context.Background(), parentLease.ID)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCleanKilledWindowsSupervisor$")
	cmd.Env = append(os.Environ(), "MYENV_LEASE_TREE_DATABASE="+database)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatal("missing lease handshake")
	}
	leaseID := scanner.Text()
	if !scanner.Scan() || scanner.Text() != "tree-ready" {
		t.Fatal("tree not started")
	}
	publish(2)
	publish(3)
	service := &Service{}
	livePreview, err := service.Clean(ctx, project, true, nil)
	if err != nil || livePreview.Candidates != 0 || livePreview.RecoverableLeases != 0 || livePreview.Changed {
		t.Fatalf("live preview %+v %v", livePreview, err)
	}
	live, err := service.Clean(ctx, project, false, nil)
	if err != nil || live.Removed != 0 || live.RecoveredLeases != 0 {
		t.Fatalf("live protection %+v %v", live, err)
	}
	if err = cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	if err == nil {
		t.Fatal("supervisor not killed")
	}
	rows, err := s.LeaseRecords(ctx, "", 128)
	if err != nil || len(rows) != 2 {
		t.Fatalf("lease missing %+v %v", rows, err)
	}
	var childLease state.LeaseRecord
	for _, row := range rows {
		if row.ID == leaseID {
			childLease = row
		}
	}
	if childLease.ID == "" {
		t.Fatal("child lease missing")
	}
	for {
		gone, err := windowsLeaseReclaimable(childLease)
		if err != nil {
			t.Fatal(err)
		}
		if gone {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("job remained after supervisor exit")
		case <-time.After(10 * time.Millisecond):
		}
	}
	mixed, err := service.Clean(ctx, project, true, nil)
	if err != nil || mixed.RecoverableLeases != 1 || mixed.Candidates != 0 || mixed.Changed {
		t.Fatalf("mixed lease preview %+v %v", mixed, err)
	}
	if err = s.ReleaseLease(ctx, parentLease.ID); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatal(err)
	}
	var previewIDs []string
	preview, err := service.Clean(ctx, project, true, func(item CleanItem) error { previewIDs = append(previewIDs, item.ID); return nil })
	if err != nil || preview.Candidates != 1 || preview.RecoverableLeases != 1 || preview.RecoveredLeases != 0 || preview.Removed != 0 || preview.Changed {
		t.Fatalf("dead preview %+v %v", preview, err)
	}
	after, err := os.ReadFile(database)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("preview mutated database", err)
	}
	rows, err = s.LeaseRecords(ctx, "", 128)
	if err != nil || len(rows) != 1 {
		t.Fatal("preview removed lease", err)
	}
	var removedIDs []string
	recovered, err := service.Clean(ctx, project, false, func(item CleanItem) error { removedIDs = append(removedIDs, item.ID); return nil })
	if err != nil || recovered.RecoveredLeases != 1 || recovered.Removed != 1 {
		t.Fatalf("recovery %+v %v", recovered, err)
	}
	if len(previewIDs) != 1 || len(removedIDs) != 1 || previewIDs[0] != removedIDs[0] {
		t.Fatalf("preview %v differs from execution %v", previewIDs, removedIDs)
	}
	rows, err = s.LeaseRecords(ctx, "", 128)
	if err != nil || len(rows) != 0 {
		t.Fatal("recovered lease remains", err)
	}
}
