package cli

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"fmt"
	"myenv/internal/backend"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTUIContinuousSelectionAndEditing(t *testing.T) {
	root := t.TempDir()
	c := NewUIController(t.TempDir())
	defer c.Close()
	m := &tuiModel{controller: c, width: 80, height: 24}
	m.switchProject(root, false)
	waitUITask(t, c)
	m.Update(tuiUpdate{})
	m.mode = "maintenance"
	m.menuIndex = 6
	beforeTask := c.Current().ID
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if c.Current().ID != beforeTask || !strings.Contains(m.message, "用户默认") {
		t.Fatal("default tool action escaped project scope")
	}
	m.mode = ""
	m.message = ""
	// Actual catalog-shaped fixture tests the UI boundary, not network resolution.
	rows := VersionResult{Releases: []backend.CatalogRelease{{Tool: "java", Version: "jdk8u504-b01", Provider: "Eclipse Temurin", URL: "https://github.com/adoptium/temurin8-binaries/releases/download/jdk8u504-b01/package.zip"}}}
	data, _ := json.Marshal(rows)
	c.mu.Lock()
	c.tasks = append(c.tasks, UITask{ID: 100, State: "succeeded", Request: UIRequest{Directory: root, Tool: "java", Action: "versions"}, Result: data})
	c.mu.Unlock()
	m.Update(tuiUpdate{})
	if !m.versionMode || m.mode != "" {
		t.Fatal("query did not directly show versions")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != "save" {
		t.Fatal("selection did not lead to save confirmation")
	}
	if _, e := os.Stat(filepath.Join(root, "myenv.yaml")); !os.IsNotExist(e) {
		t.Fatal("selection wrote config")
	}
	// The last catalog entry must remain visible in both accepted terminal sizes.
	for i := 0; i < 35; i++ {
		m.catalog.Releases = append(m.catalog.Releases, backend.CatalogRelease{Version: fmt.Sprintf("entry-%02d", i), URL: "https://example.test/archive"})
	}
	for _, size := range [][2]int{{80, 24}, {120, 32}} {
		m.width, m.height = size[0], size[1]
		m.mode = ""
		m.versionMode = true
		m.versionIndex = len(m.catalog.Releases) - 1
		if !strings.Contains(m.View().Content, "entry-34") {
			t.Fatal("selected last version clipped")
		}
	}
	m.mode = "save"
	m.versionMode = false
	m.filter = "504"
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m.versionMode || m.filter != "504" {
		t.Fatal("back lost filter")
	}
	for _, size := range [][2]int{{80, 24}, {120, 32}} {
		m.width, m.height = size[0], size[1]
		for _, mode := range []string{"save", "command", "maintenance", "preview", "task"} {
			m.mode = mode
			m.versionMode = false
			if !strings.Contains(m.View().Content, "任务") {
				t.Fatal("task clipped")
			}
		}
	}
	m.mode = "command"
	m.runArgs = []string{"java", "-version"}
	m.beginEdit(6, "中文 路径")
	m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	m.Update(tea.PasteMsg{Content: "文 新"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.runArgs[0] != "中文 新 路" {
		t.Fatalf("cursor/paste result %q", m.runArgs[0])
	}
	m.beginEdit(6, m.runArgs[0])
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m.Update(tea.PasteMsg{Content: "java"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.runArgs[0] != "java" {
		t.Fatal("clear/paste failed")
	}
}

func TestTUIFirstPythonAndCanceledArgument(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 32}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			c := NewUIController(t.TempDir())
			defer c.Close()
			m := &tuiModel{controller: c, width: size[0], height: size[1]}
			root := t.TempDir()
			m.switchProject(root, false)
			waitUITask(t, c)
			m.Update(tuiUpdate{})
			before := c.Current().ID
			m.homeIndex = 0
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if m.mode != "version-setup" || c.Current().ID != before {
				t.Fatal("Python entry queried before source selection")
			}
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			failed := waitUITask(t, c)
			m.Update(tuiUpdate{})
			if failed.State != "failed" || m.mode != "version-setup" {
				t.Fatalf("missing uv: %+v mode=%s", failed.Error, m.mode)
			}
			m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
			if m.request.Provider != "python.org" {
				t.Fatal("source inaccessible after failure")
			}
			m.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
			m.Update(tea.PasteMsg{Content: "3.12"})
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if m.mode != "save" || m.request.Selection != "python@3.12" || m.request.Provider != "python.org" {
				t.Fatal("manual version did not preserve source or reach confirmation")
			}
			if _, err := os.Stat(filepath.Join(root, "myenv.yaml")); !os.IsNotExist(err) {
				t.Fatal("version choice wrote config")
			}
			// The bootstrap-free manual Astral path reaches a real whole-environment preview.
			m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
			m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
			m.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
			m.Update(tea.PasteMsg{Content: "3.12"})
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			initialized := waitUITask(t, c)
			m.Update(tuiUpdate{})
			if initialized.State != "succeeded" {
				t.Fatalf("explicit init: %+v", initialized.Error)
			}
			plan := waitUITask(t, c)
			m.Update(tuiUpdate{})
			if plan.State != "succeeded" || m.mode != "preview" || plan.Request.Provider != "astral" {
				t.Fatalf("first-use preview: %+v mode=%s", plan.Error, m.mode)
			}
			if m.workbench == nil || len(m.workbench.Applied) != 0 {
				t.Fatal("preview installed environment")
			}
			m.mode = "command"
			m.runArgs = []string{"java", "-version"}
			m.menuIndex = 2
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			m.Update(tea.PasteMsg{Content: "取消的参数"})
			m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
			if len(m.runArgs) != 2 {
				t.Fatalf("cancel mutated args: %q", m.runArgs)
			}
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			m.Update(tea.PasteMsg{Content: "中文 空格"})
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if len(m.runArgs) != 3 || m.runArgs[2] != "中文 空格" {
				t.Fatalf("confirmed append: %q", m.runArgs)
			}
			m.beginEdit(8, "起点"+strings.Repeat("中文 argument ", 400)+"终点")
			for _, key := range []rune{tea.KeyHome, tea.KeyEnd, tea.KeyHome} {
				m.Update(tea.KeyPressMsg{Code: key})
				view := m.View().Content
				if !strings.Contains(view, "▏") {
					t.Fatalf("cursor hidden after %v", key)
				}
				if key == tea.KeyHome && !strings.Contains(view, "▏起点") {
					t.Fatal("Home viewport missing start")
				}
				if key == tea.KeyEnd && !strings.Contains(view, "终点▏") {
					t.Fatal("End viewport missing end")
				}
			}
		})
	}
}

func TestTUIAllCatalogFailuresRecover(t *testing.T) {
	for _, tool := range uiTools {
		t.Run(tool, func(t *testing.T) {
			c := NewUIController(t.TempDir())
			defer c.Close()
			m := &tuiModel{controller: c, width: 80, height: 24}
			m.switchProject(t.TempDir(), false)
			waitUITask(t, c)
			m.Update(tuiUpdate{})
			for i, v := range uiTools {
				if v == tool {
					m.homeIndex = i
				}
			}
			before := c.Current().ID
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if m.mode != "version-setup" || c.Current().ID != before {
				t.Fatal("entry did not offer query/manual choice")
			}
			m.request.Certificate = filepath.Join(t.TempDir(), "missing-ca.pem")
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			failed := waitUITask(t, c)
			m.Update(tuiUpdate{})
			if failed.State != "failed" || m.mode != "version-setup" {
				t.Fatalf("failure has no normal recovery: %+v mode=%s", failed.Error, m.mode)
			}
			m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			m.Update(tea.PasteMsg{Content: "17"})
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if m.mode != "save" || m.request.Selection != tool+"@17" {
				t.Fatal("manual entry unreachable")
			}
			m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
			if m.mode != "version-setup" {
				t.Fatal("back lost recovery context")
			}
		})
	}
}

func TestTUIQueryCancellationKeepsSelectionContext(t *testing.T) {
	c := NewUIController(t.TempDir())
	defer c.Close()
	m := &tuiModel{controller: c, width: 80, height: 24, mode: "task", request: UIRequest{Directory: t.TempDir(), Tool: "go"}}
	// A canceled query event uses the same controller snapshot as real cancellation.
	c.mu.Lock()
	c.tasks = append(c.tasks, UITask{ID: 1, State: "running", Request: UIRequest{Directory: m.request.Directory, Tool: "go", Action: "versions"}})
	c.mu.Unlock()
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.mode != "task" {
		t.Fatal("escaped query before cleanup")
	}
	c.mu.Lock()
	c.tasks[0].State = "canceled"
	c.mu.Unlock()
	m.Update(tuiUpdate{})
	if m.mode != "version-setup" || m.request.Tool != "go" {
		t.Fatal("cancel lost retry/manual context")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.editing {
		t.Fatal("manual entry unavailable after cancellation")
	}
}
