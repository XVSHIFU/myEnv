package cli

import (
	"context"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func waitUITask(t *testing.T, c *UIController) UITask {
	t.Helper()
	c.mu.Lock()
	done := c.done
	c.mu.Unlock()
	select {
	case <-done:
		return c.Current()
	case <-time.After(10 * time.Second):
		t.Fatal("task did not finish")
		return UITask{}
	}
}
func TestUITaskScopeAndBoundedHistory(t *testing.T) {
	root := t.TempDir()
	c := NewUIController(t.TempDir())
	defer c.Close()
	request := UIRequest{Directory: root, Action: "init", Selection: "java@8"}
	if _, err := c.Start(request); err != nil {
		t.Fatal(err)
	}
	request.Directory = t.TempDir()
	result := waitUITask(t, c)
	if result.State != "succeeded" {
		t.Fatalf("%+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "myenv.yaml")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 18; i++ {
		if _, err := c.Start(UIRequest{Directory: root, Action: "status"}); err != nil {
			t.Fatal(err)
		}
		waitUITask(t, c)
	}
	if len(c.tasks) != 16 {
		t.Fatal("unbounded task history")
	}
}

func TestUIBuildConfirmationCancellation(t *testing.T) {
	root := t.TempDir()
	for name, value := range map[string]string{
		"myenv.yaml":     "schema: 1\ntools: {python: '3.12'}\npython: {project: '.'}\n",
		"pyproject.toml": "[project]\nname='ui-build-test'\nversion='0.1.0'\n[build-system]\nrequires=[]\nbuild-backend='must_not_execute'\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	c := NewUIController(t.TempDir())
	defer c.Close()
	id, err := c.Start(UIRequest{Action: "sync", Directory: root})
	if err != nil {
		t.Fatal(err)
	}
	timeout := time.After(5 * time.Second)
	for c.Current().State != "confirm" {
		select {
		case <-c.Updates():
		case <-timeout:
			t.Fatal("no build confirmation")
		}
	}
	c.Confirm(id+1, true) // An old/new task must not authorize this one.
	if c.Current().State != "confirm" {
		t.Fatal("wrong task authorized")
	}
	c.Cancel(id)
	c.Close()
	if got := c.Current(); got.State != "canceled" {
		t.Fatalf("%+v", got)
	}
	if _, err := os.Stat(filepath.Join(root, "myenv.lock")); !os.IsNotExist(err) {
		t.Fatalf("unapproved build wrote lock: %v", err)
	}
}
func TestUICancelWaitsForCore(t *testing.T) {
	root := t.TempDir()
	c := NewUIController(t.TempDir())
	defer c.Close()
	os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {java: '8'}\n"), 0600)
	// The real core must wait on the same mutation lock and honor cancellation.
	if err := os.MkdirAll(filepath.Join(root, ".myenv"), 0700); err != nil {
		t.Fatal(err)
	}
	unlock, err := state.LockWorkspace(context.Background(), filepath.Join(root, ".myenv", "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	id, err := c.Start(UIRequest{Directory: root, Action: "sync"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for c.Current().Phase == "" {
		select {
		case <-c.Updates():
		case <-deadline:
			t.Fatal("core did not reach lock")
		}
	}
	c.Cancel(id)
	c.Close()
	result := c.Current()
	if result.State != "canceled" {
		t.Fatalf("%+v", result)
	}
	if _, err = c.Start(UIRequest{Directory: root, Action: "status"}); err == nil {
		t.Fatal("accepted task after close")
	}
}
