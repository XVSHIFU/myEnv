//go:build windows || linux

package core

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"myenv/internal/state"
)

func TestEmptyTrackedPreparationRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if root := os.Getenv("MYENV_TEST_EMPTY_PREPARATION_OWNER"); root != "" {
		work := filepath.Join(root, ".myenv")
		s, err := state.Open(ctx, filepath.Join(work, "state.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		for i, id := range []string{"00000000000000000000000000000001", "00000000000000000000000000000002"} {
			directory := filepath.Join(work, "generations", id)
			if i == 0 {
				err = s.BeginTrackedOperation(ctx, id, directory, "digest")
			} else {
				err = s.BeginGuardedOperation(ctx, id, directory, "digest")
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		live, err := (&Service{}).Clean(ctx, root, true, nil)
		if err != nil || live.Candidates != 0 {
			t.Fatalf("live owner preview: %+v %v", live, err)
		}
		return
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".myenv"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {node: '22'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, executable, "-test.run=^TestEmptyTrackedPreparationRecovery$")
	command.Env = append(os.Environ(), "MYENV_TEST_EMPTY_PREPARATION_OWNER="+root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("owner: %v %s", err, output)
	}
	database := filepath.Join(root, ".myenv", "state.db")
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := (&Service{}).Clean(ctx, root, true, nil)
	if err != nil || preview.Candidates != 1 || preview.Changed {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	after, err := os.ReadFile(database)
	if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("preview changed state", err)
	}
	actual, err := (&Service{}).Clean(ctx, root, false, nil)
	if err != nil || actual.RecoveredPreparations != 1 || actual.Removed != 1 || actual.Bytes != preview.Bytes {
		t.Fatalf("actual: %+v %v", actual, err)
	}
	repeat, err := (&Service{}).Clean(ctx, root, false, nil)
	if err != nil || repeat.Changed || repeat.Removed != 0 {
		t.Fatalf("historical protection: %+v %v", repeat, err)
	}
}
