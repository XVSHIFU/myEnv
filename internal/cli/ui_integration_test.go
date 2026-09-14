package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUIRetainedJavaWorkflow(t *testing.T) {
	namespace := os.Getenv("MYENV_UI_TEST_NAMESPACE")
	if namespace == "" {
		t.Skip("requires isolated retained Java 8 cache")
	}
	directory := filepath.Join(t.TempDir(), "中文 项目")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	c := NewUIController(namespace)
	defer c.Close()
	for _, r := range []UIRequest{
		{Action: "init", Selection: "java@8"},
		{Action: "desired", Selection: "java@8"},
		{Action: "sync", Preview: true},
		{Action: "sync"},
		{Action: "sync", Locked: true},
		{Action: "doctor", Deep: true},
		{Action: "clean"},
	} {
		r.Directory = directory
		if _, err := c.Start(r); err != nil {
			t.Fatal(err)
		}
		c.mu.Lock()
		done := c.done
		c.mu.Unlock()
		select {
		case <-done:
		case <-time.After(90 * time.Second):
			c.Cancel(c.Current().ID)
			t.Fatal("task timeout")
		}
		if got := c.Current(); got.State != "succeeded" {
			t.Fatalf("%s: %+v", r.Action, got)
		}
		if r.Action == "desired" && c.Current().View.Status.Generation != nil {
			t.Fatal("desired preview applied an environment")
		}
	}
	var out bytes.Buffer
	code, err := RunCommand(context.Background(), RunRequest{Directory: directory, Namespace: namespace, Stdin: bytes.NewReader(nil), Stdout: &out, Stderr: &out}, []string{"java", "-version"})
	if err != nil || code != 0 || !bytes.Contains(out.Bytes(), []byte("1.8.0")) {
		t.Fatalf("code=%d err=%v %s", code, err, out.String())
	}
	t.Log("real init/desired preview/sync preview/sync/locked sync/deep doctor/clean preview/run completed in Chinese directory")
}
