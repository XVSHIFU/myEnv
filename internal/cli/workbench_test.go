package cli

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/charmbracelet/x/ansi"
	"myenv/internal/config"
	"myenv/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkbenchProjectIsolationAndLayout(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "甲", "同名")
	b := filepath.Join(root, "乙", "同名")
	for _, p := range []string{a, b} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	c := NewUIController(t.TempDir())
	defer c.Close()
	m := &tuiModel{controller: c}
	m.switchProject(a, false)
	if m.message != "" {
		t.Fatal(m.message)
	}
	waitUITask(t, c)
	m.Update(tuiUpdate{})
	if m.workbench == nil || !m.workbench.Empty {
		t.Fatal("missing empty state")
	}
	entries, _ := os.ReadDir(a)
	if len(entries) != 0 {
		t.Fatal("selection initialized project")
	}
	m.runArgs = []string{"中文", "with space"}
	m.request.Selection = "java@8"
	m.switchProject(b, false)
	waitUITask(t, c)
	if m.request.Selection != "" {
		t.Fatal("draft leaked across equal basenames")
	}
	m.switchProject(filepath.Join(a, "."), false)
	waitUITask(t, c)
	m.Update(tuiUpdate{})
	if m.request.Selection != "java@8" || m.runArgs[0] != "中文" {
		t.Fatal("draft not restored")
	}
	for _, size := range [][2]int{{80, 24}, {120, 32}} {
		m.width, m.height = size[0], size[1]
		for _, mode := range []string{"", "menu", "theme", "source", "history", "projects"} {
			m.mode = mode
			v := m.View()
			lines := strings.Split(v.Content, "\n")
			if len(lines) != size[1] {
				t.Fatalf("%s height %d", mode, len(lines))
			}
			for _, line := range lines {
				if ansi.StringWidth(line) != size[0] {
					t.Fatalf("%s width %d: %q", mode, ansi.StringWidth(line), line)
				}
			}
			if mode == "" && !strings.Contains(v.Content, "运行命令") {
				t.Fatal("clipped operation")
			}
		}
	}
	m.theme = "system"
	if v := m.View(); v.ForegroundColor != nil || v.BackgroundColor != nil {
		t.Fatal("system theme overrides terminal palette")
	}
	m.mode = "theme"
	m.menuIndex = 2
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.theme != "dark" {
		t.Fatal("theme selection")
	}
}
func TestWorkbenchDefaultPythonSourceAndPreview(t *testing.T) {
	namespace := t.TempDir()
	_, err := QueryVersions(context.Background(), VersionQuery{Tool: "python", Namespace: namespace})
	if err == nil || !strings.Contains(err.Error(), "Astral") {
		t.Fatalf("default query did not use Astral: %v", err)
	}
	entries, _ := os.ReadDir(namespace)
	if len(entries) != 0 {
		t.Fatal("query installed a backend")
	}
	root := t.TempDir()
	_, _, err = config.InitWithInput(root, func(string, string) (string, string, error) { return "python", "3.12", nil })
	if err != nil {
		t.Fatal(err)
	}
	service := &core.Service{UserConfigDirectory: namespace}
	result, err := service.UseWithRequest(context.Background(), core.UseRequest{Directory: root, Selection: "python@astral/3.13", Preview: true})
	if err != nil || result.Changed || !result.DeclarationChanged || result.Plan == nil || result.Plan.UnresolvedPython != "astral/3.13" {
		t.Fatalf("preview: %+v %v", result, err)
	}
	w, err := service.Workbench(context.Background(), root)
	if err != nil || w.Desired["python"] != "astral/3.13" || len(w.Applied) != 0 {
		t.Fatalf("workbench: %+v %v", w, err)
	}
	if _, err := os.Stat(filepath.Join(root, "myenv.lock")); !os.IsNotExist(err) {
		t.Fatal("preview wrote lock")
	}
	_, err = config.SetTool(filepath.Join(root, "myenv.yaml"), "python", "3.12")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Sync(context.Background(), core.SyncRequest{Directory: root, ExpectedDigest: w.Digest})
	if err == nil || !strings.Contains(err.Error(), "INPUT_CHANGED") {
		t.Fatalf("stale preview accepted: %v", err)
	}
}
