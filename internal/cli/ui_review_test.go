package cli

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"myenv/internal/config"
	"myenv/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTUIReviewInteractions(t *testing.T) {
	c := NewUIController(t.TempDir())
	defer c.Close()
	m := &tuiModel{controller: c, request: UIRequest{Tool: "java"}, runArgs: []string{"java", "-version"}, homeIndex: 2, width: 80, height: 24}
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.request.Tool != "java" || m.homeIndex != 3 {
		t.Fatal("navigation changed selected language")
	}
	m.selectTool("go")
	if strings.Join(m.runArgs, " ") != "go version" || !strings.Contains(m.View().Content, `["go","version"]`) {
		t.Fatal("selected tool command missing")
	}
	m.runArgs = []string{"go", "test", "中文 包"}
	m.selectTool("python")
	m.selectTool("go")
	if m.runArgs[2] != "中文 包" {
		t.Fatal("custom command lost")
	}
	for i, key := range []string{"l", "d"} {
		m.mode = "maintenance"
		m.menuIndex = i
		before := m.View().Content
		m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
		if before == m.View().Content {
			t.Fatalf("invisible %s option", key)
		}
	}
	for _, mode := range []string{"menu", "theme", "source", "history", "preview", "run"} {
		m.mode = mode
		m.quit = false
		_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if !m.quit || cmd == nil {
			t.Fatalf("Ctrl+C swallowed in %s", mode)
		}
	}
}

func TestTUIRepairAfterPreviewReal(t *testing.T) {
	ns := os.Getenv("MYENV_UI_TEST_NAMESPACE")
	if ns == "" {
		t.Skip("requires isolated retained Java8 cache")
	}
	root := t.TempDir()
	_, _, err := config.InitWithInput(root, func(string, string) (string, string, error) { return "java", "8", nil })
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the verified lock as well as the archive: this regression concerns
	// request scope, not today's network catalog availability.
	lock, err := os.ReadFile(filepath.Join(os.Getenv("MYENV_TUI_NATIVE_PROJECT"), "myenv.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "myenv.lock"), lock, 0600); err != nil {
		t.Fatal(err)
	}
	c := NewUIController(ns)
	defer c.Close()
	m := &tuiModel{controller: c, request: UIRequest{Directory: root, Tool: "java", Preview: true}, mode: "maintenance", menuIndex: 0, width: 80, height: 24}
	wait := func() UITask {
		c.mu.Lock()
		done := c.done
		c.mu.Unlock()
		select {
		case <-done:
			return c.Current()
		case <-time.After(90 * time.Second):
			c.Cancel(c.Current().ID)
			t.Fatal("timeout")
			return UITask{}
		}
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	first := wait()
	if first.State != "succeeded" {
		t.Fatalf("preview: %+v", first)
	}
	m.Update(tuiUpdate{})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m.mode = "maintenance"
	m.menuIndex = 2
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	wait()
	m.Update(tuiUpdate{})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	second := wait()
	var result core.SyncResult
	_ = json.Unmarshal(second.Result, &result)
	if second.State != "succeeded" || second.Request.Preview || result.Plan != nil || !result.Changed || result.Generation == nil {
		t.Fatalf("repair did not really apply: %+v %s", second, second.Result)
	}
	t.Log("sync preview followed by repair produced a real generation, not a plan")
}

func TestUISummaryPreviewAndCleaning(t *testing.T) {
	task := UITask{State: "succeeded", Request: UIRequest{Action: "sync", Preview: true}, Result: json.RawMessage(`{"plan":{"needs_apply":true,"tools":{"java":{"version":"jdk-17","backend":"java-official-v1"}}}}`), View: &core.Workbench{Applied: map[string]config.RuntimeLock{"java": {Version: "jdk-8", Backend: "java-official-v1"}}, Desired: map[string]string{"java": "17"}}}
	s := summarizeTask(task)
	if len(s.Changes) != 1 || s.Changes[0].From != "jdk-8" || s.Changes[0].To != "jdk-17" {
		t.Fatalf("wrong transition %+v", s)
	}
	task.Request.Action = "clean"
	task.Result = json.RawMessage(`{"candidates":2,"bytes":1048576,"unknown_leases":1}`)
	s = summarizeTask(task)
	if !strings.Contains(strings.Join(s.Lines, " "), "1.00 MiB") || !strings.Contains(strings.Join(s.Lines, " "), "继续保留保护") {
		t.Fatalf("missing clean evidence %+v", s)
	}
}
