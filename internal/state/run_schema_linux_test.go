package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myenv/internal/runner"
)

func TestOpenForRunCleanOverlap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	root := t.TempDir()
	path := filepath.Join(root, "state.db")
	writer, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if err := writer.Publish(ctx, Generation{ID: "one", Directory: root, InputDigest: "one", NodeExecutable: "/bin/sh"}, ""); err != nil {
		t.Fatal(err)
	}
	selected, err := OpenForRun(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer selected.Close()
	lease, err := selected.AcquireActiveWithCompletion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	done := make(chan error, 1)
	go func() {
		code, err := runner.Execute(ctx, runner.Process{Executable: "/bin/sh", Args: []string{"-c", "printf ready > started; read value"}, Directory: root, Stdin: r, Environment: os.Environ()})
		if err == nil && code != 0 {
			err = fmt.Errorf("user command exited %d", code)
		}
		done <- err
	}()
	finished := false
	defer func() {
		if !finished {
			cancel()
			w.Close()
			<-done
		}
	}()
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("user command not ready")
		}
		time.Sleep(5 * time.Millisecond)
	}
	previous := "one"
	for _, id := range []string{"two", "three"} {
		if err := writer.Publish(ctx, Generation{ID: id, Directory: filepath.Join(root, id), InputDigest: id, NodeExecutable: "/bin/sh"}, previous); err != nil {
			t.Fatal(err)
		}
		previous = id
	}
	for i := 0; i < 3; i++ {
		if marked, err := writer.MarkDeleting(ctx, "one", root); err != nil || marked {
			t.Fatal("clean reserved running generation", marked, err)
		}
	}
	if _, err := w.WriteString("done\n"); err != nil {
		t.Fatal(err)
	}
	w.Close()
	runErr := <-done
	finished = true
	if runErr != nil {
		t.Fatal(runErr)
	}
	if err := selected.ReleaseLease(ctx, lease.ID); err != nil {
		t.Fatal(err)
	}
	if marked, err := writer.MarkDeleting(ctx, "one", root); err != nil || !marked {
		t.Fatal("completed generation not reclaimable", marked, err)
	}
}
