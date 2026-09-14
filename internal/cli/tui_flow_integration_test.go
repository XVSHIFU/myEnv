package cli

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTUIContinuousRealWorkflow(t *testing.T) {
	ns := os.Getenv("MYENV_UI_TEST_NAMESPACE")
	if ns == "" {
		t.Skip("requires retained Java 8/21 archives")
	}
	root := os.Getenv("MYENV_UI_FLOW_PROJECT")
	if root == "" {
		t.Fatal("explicit isolated evidence directory required")
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	c := NewUIController(ns)
	defer c.Close()
	m := &tuiModel{controller: c, width: 80, height: 24}
	m.switchProject(root, false)
	wait := func() UITask {
		t.Helper()
		c.mu.Lock()
		done := c.done
		c.mu.Unlock()
		select {
		case <-done:
		case <-time.After(90 * time.Second):
			c.Cancel(c.Current().ID)
			t.Fatal("task timeout")
		}
		got := c.Current()
		m.Update(tuiUpdate{})
		return got
	}
	enter := func() { m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) }
	wait()
	for _, step := range []struct {
		major   int
		version string
	}{{8, "jdk8u504-b01"}, {21, "jdk-21.0.12.1+1"}} {
		m.mode = ""
		m.homeIndex = 2
		m.request.Major = step.major
		enter()
		enter()
		got := wait()
		if got.State != "succeeded" || !m.versionMode {
			t.Fatalf("versions: %+v", got)
		}
		rows := m.versionRows()
		found := false
		for i, r := range rows {
			if r.Version == step.version {
				m.versionIndex = i
				found = true
				break
			}
		}
		if !found {
			t.Fatal("retained version absent in real catalog")
		}
		before, _ := os.ReadFile(filepath.Join(root, "myenv.yaml"))
		enter()
		after, _ := os.ReadFile(filepath.Join(root, "myenv.yaml"))
		if !bytes.Equal(before, after) || m.mode != "save" {
			t.Fatal("selection wrote configuration")
		}
		enter()
		got = wait()
		if got.Request.Action == "init" {
			got = wait()
		}
		if got.State != "succeeded" || m.mode != "preview" {
			t.Fatalf("save preview: %+v", got)
		}
		if step.major == 21 && got.View.Applied["java"].Version != "jdk8u504-b01" {
			t.Fatal("preview changed active version")
		}
		enter()
		got = wait()
		if got.State != "succeeded" || got.View.Applied["java"].Version != step.version {
			t.Fatalf("sync: %+v", got)
		}
		t.Logf("real UI selection/save/preview/confirm installed %s; task #%d", step.version, got.ID)
	}
	var out bytes.Buffer
	code, e := RunCommand(context.Background(), RunRequest{Directory: root, Namespace: ns, Stdin: strings.NewReader(""), Stdout: &out, Stderr: &out}, []string{"java", "-version"})
	if e != nil || code != 0 || !strings.Contains(out.String(), "21.0.12.1") {
		t.Fatalf("run %d %v %s", code, e, out.String())
	}
	t.Logf("run exit %d: %s", code, out.String())
	// A real stale-input rejection must preserve the old environment and expose why.
	r := m.request
	r.Action = "sync"
	r.Preview = true
	m.startFlow(r)
	got := wait()
	if got.State != "succeeded" {
		t.Fatal(got.Error)
	}
	m.previewDigest = "changed-after-preview"
	enter()
	got = wait()
	if got.State != "failed" || got.Error == nil || got.View.Applied["java"].Version != "jdk-21.0.12.1+1" {
		t.Fatalf("stale input protection: %+v", got)
	}
	t.Logf("failure shown, old version retained: %s; %s", got.Error.Message, got.Error.NextAction)
}
