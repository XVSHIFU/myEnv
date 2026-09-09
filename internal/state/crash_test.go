package state

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// The child holds the real OS lock and a committed preparing record until killed.
func TestRecoveryKilledOwner(t *testing.T) {
	if root := os.Getenv("MYENV_STATE_CRASH_CHILD"); root != "" {
		ctx := context.Background()
		unlock, err := LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
		if err != nil {
			t.Fatal(err)
		}
		defer unlock()
		s, err := Open(ctx, filepath.Join(root, "state.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		if err = s.BeginOperation(ctx, "interrupted", filepath.Join(root, "pending"), "pending-digest"); err != nil {
			t.Fatal(err)
		}
		if err = s.BeginGuardedOperation(ctx, "uncertain", filepath.Join(root, "uncertain"), "pending-digest"); err != nil {
			t.Fatal(err)
		}
		if err = s.BeginGuardedOperation(ctx, "settled", filepath.Join(root, "settled"), "pending-digest"); err != nil {
			t.Fatal(err)
		}
		if err = s.ConfirmOperationTreesDone(ctx, "settled"); err != nil {
			t.Fatal(err)
		}
		fmt.Println("operation-ready")
		for {
			time.Sleep(time.Second)
		}
	}
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s, err := Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	g := Generation{ID: "active", Directory: filepath.Join(root, "active"), InputDigest: "active-digest", NodeExecutable: filepath.Join(root, "active", "node")}
	if err = s.Publish(ctx, g, ""); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRecoveryKilledOwner$")
	cmd.Env = append(os.Environ(), "MYENV_STATE_CRASH_CHILD="+root)
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
	if scanner := bufio.NewScanner(stdout); !scanner.Scan() || scanner.Text() != "operation-ready" {
		t.Fatal("child failed before preparing operation")
	}
	contended, stop := context.WithTimeout(ctx, 150*time.Millisecond)
	unlock, err := LockWorkspace(contended, filepath.Join(root, "modify.lock"))
	stop()
	if err == nil {
		unlock()
		t.Fatal("live owner did not exclude second writer")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if err = cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	if err == nil {
		t.Fatal("child was not terminated")
	}
	unlock, err = LockWorkspace(ctx, filepath.Join(root, "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	s, err = Open(ctx, filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	count, err := s.RecoverInterrupted(ctx)
	if err != nil || count != 1 {
		t.Fatalf("crash recovery count %d: %v", count, err)
	}
	active, err := s.Active(ctx)
	if err != nil || active == nil || *active != g {
		t.Fatalf("recovery changed active: %+v %v", active, err)
	}
	var status string
	if err = s.db.QueryRowContext(ctx, `SELECT status FROM operations WHERE id='settled'`).Scan(&status); err != nil || status != "failed" {
		t.Fatalf("confirmed operation status %q %v", status, err)
	}
	if err = s.db.QueryRowContext(ctx, `SELECT status FROM operations WHERE id='interrupted'`).Scan(&status); err != nil || status != "preparing" {
		t.Fatalf("interrupted status %q %v", status, err)
	}
	if count, err = s.RecoverInterrupted(ctx); err != nil || count != 0 {
		t.Fatalf("recovery not idempotent %d %v", count, err)
	}
	if err = s.db.QueryRowContext(ctx, `SELECT status FROM operations WHERE id='uncertain'`).Scan(&status); err != nil || status != "preparing" {
		t.Fatalf("killed owner's uncertain tree lost protection: %q %v", status, err)
	}
	if marked, err := s.MarkPreparationDeleting(ctx, "uncertain", filepath.Join(root, "uncertain")); err != nil || marked {
		t.Fatalf("reserved killed owner's uncertain tree: %v %v", marked, err)
	}
}
