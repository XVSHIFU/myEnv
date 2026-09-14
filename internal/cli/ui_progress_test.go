package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"myenv/internal/backend"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestUIProgressTimestampsAndTerminalImmunity(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "myenv.yaml"), []byte("schema: 1\ntools: {java: '8'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".myenv"), 0700); err != nil {
		t.Fatal(err)
	}
	unlock, err := state.LockWorkspace(context.Background(), filepath.Join(root, ".myenv", "modify.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	c := NewUIController(t.TempDir())
	defer c.Close()
	before := time.Now().UnixMilli()
	id, err := c.Start(UIRequest{Directory: root, Action: "sync"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for c.Current().Phase != "waiting" {
		select {
		case <-c.Updates():
		case <-deadline:
			t.Fatal("core did not reach mutation lock")
		}
	}
	running := c.Current()
	if running.StartedAt < before || running.UpdatedAt < running.StartedAt || running.FinishedAt != 0 || !reflect.DeepEqual(running.Phases, []string{"waiting"}) {
		t.Fatalf("wrong running timestamps/phases: %+v", running)
	}
	c.download(id, backend.DownloadProgress{ID: 1, Label: "archive.zip", Completed: 7, Total: 100})
	c.Cancel(id)
	if canceling := c.Current(); canceling.State != "canceling" || canceling.Progress != nil {
		t.Fatalf("cancel retained progress: %+v", canceling)
	}
	finished := waitUITask(t, c)
	if finished.State != "canceled" || finished.FinishedAt < finished.StartedAt || finished.UpdatedAt != finished.FinishedAt || finished.Progress != nil {
		t.Fatalf("wrong terminal timestamps: %+v", finished)
	}
	c.phase(id, "running", "publishing")
	c.download(id, backend.DownloadProgress{ID: 2, Label: "late.zip", Completed: 90, Total: 100})
	_, _ = (uiLog{c, id}).Write([]byte("late child output"))
	if got := c.Current(); !reflect.DeepEqual(got, finished) {
		t.Fatalf("terminal task changed after late callback: %+v", got)
	}
}

func TestUIProgressStreamingCoalescesAndClears(t *testing.T) {
	c := NewUIController(t.TempDir())
	c.tasks = []UITask{{ID: 2, State: "running", StartedAt: time.Now().UnixMilli()}}
	defer c.Close()
	c.mu.Lock()
	c.notify()
	c.mu.Unlock()
	<-c.Updates()
	writer := uiLog{c, 2}
	for i := 0; i < 100; i++ {
		if n, err := writer.Write([]byte("streamed child output\n")); err != nil || n != len("streamed child output\n") {
			t.Fatal("stream writer rejected bytes")
		}
		c.download(2, backend.DownloadProgress{ID: 1, Label: "node.zip", Completed: int64(i), Total: 100})
	}
	select {
	case <-c.Updates():
	case <-time.After(time.Second):
		t.Fatal("streaming log/progress did not notify before task completion")
	}
	snapshot := c.Current()
	if snapshot.Progress == nil || snapshot.Progress.Completed != 99 || !strings.Contains(snapshot.Log, "streamed child output") || snapshot.UpdatedAt < snapshot.StartedAt {
		t.Fatalf("notified without latest streaming data: %+v", snapshot)
	}
	c.mu.Lock()
	if c.notifyTimer != nil {
		t.Error("stream timer remained active without new data")
	}
	c.mu.Unlock()
	select {
	case <-c.Updates():
		t.Fatal("burst emitted uncoalesced updates")
	default:
	}
	c.download(2, backend.DownloadProgress{ID: 1, Done: true})
	if c.Current().Progress != nil {
		t.Fatal("download completion did not clear transfer")
	}
	c.download(2, backend.DownloadProgress{ID: 1, Label: "late.zip", Completed: 100})
	if c.Current().Progress != nil {
		t.Fatal("finished transfer revived by late bytes")
	}
	c.download(2, backend.DownloadProgress{ID: 2, Label: "sdk.zip", Completed: 10})
	c.phase(2, "running", "verifying")
	c.download(2, backend.DownloadProgress{ID: 2, Label: "old phase.zip", Completed: 20})
	if got := c.Current(); got.Progress != nil || got.Phase != "verifying" {
		t.Fatalf("phase retained stale transfer: %+v", got)
	}
	c.phase(1, "running", "publishing")
	c.download(1, backend.DownloadProgress{ID: 3, Label: "old task.zip"})
	_, _ = (uiLog{c, 1}).Write([]byte("old task output"))
	if got := c.Current(); got.Phase != "verifying" || got.Progress != nil || strings.Contains(got.Log, "old task output") {
		t.Fatalf("old task mutated current task: %+v", got)
	}
}

func TestUIProgressBoundedSnapshotsAndLogNotification(t *testing.T) {
	c := NewUIController(t.TempDir())
	c.tasks = []UITask{{ID: 1, State: "running"}}
	defer c.Close()
	writer := uiLog{c, 1}
	payload := []byte(strings.Repeat("x", 70*1024))
	if n, err := writer.Write(payload); err != nil || n != len(payload) {
		t.Fatalf("bounded log blocked/rejected child bytes: %d %v", n, err)
	}
	select {
	case <-c.Updates():
	case <-time.After(time.Second):
		t.Fatal("log-only write did not notify")
	}
	if got := c.Current(); len(got.Log) != 64*1024 || !got.LogTruncated {
		t.Fatalf("unbounded log: %d, truncated=%t", len(got.Log), got.LogTruncated)
	}
	_, _ = writer.Write([]byte("already full"))
	select {
	case <-c.Updates():
		t.Fatal("unchanged truncated log emitted another update")
	default:
	}
	for i := 0; i < 30; i++ {
		c.phase(1, "running", fmt.Sprintf("phase-%d", i))
	}
	c.download(1, backend.DownloadProgress{ID: 1, Label: "archive.zip", Completed: 4})
	snapshot := c.Current()
	history := c.History()
	if len(snapshot.Phases) != 16 || snapshot.Progress == nil || len(history) != 1 {
		t.Fatalf("wrong bounded snapshot: %+v", snapshot)
	}
	snapshot.Phases[0], history[0].Phases[1] = "modified", "modified"
	snapshot.Progress.Completed, history[0].Progress.Label = 999, "modified"
	got := c.Current()
	if got.Phases[0] != "phase-0" || got.Phases[1] != "phase-1" || got.Progress.Completed != 4 || got.Progress.Label != "archive.zip" {
		t.Fatal("snapshot aliases mutable task state")
	}
	encoded, err := json.Marshal(got)
	if err != nil || strings.Contains(string(encoded), `"total"`) || strings.Contains(string(encoded), `"finishedAt"`) {
		t.Fatalf("unknown total/unfinished time should be absent: %s %v", encoded, err)
	}
	c.phase(1, "running", "verifying") // Flush the pending event without an idle timer.
}
