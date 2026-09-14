package cli

import (
	"encoding/json"
	"myenv/internal/config"
	"myenv/internal/core"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTUITerminalEventContinuation(t *testing.T) {
	for _, action := range []string{"clean", "sync", "external"} {
		for _, size := range [][2]int{{80, 24}, {120, 32}} {
			c := NewUIController(t.TempDir())
			r := UIRequest{Directory: t.TempDir(), Action: action, Preview: action == "sync"}
			c.tasks = []UITask{{ID: 1, Request: r, State: "succeeded"}}
			m := &tuiModel{controller: c, request: r, mode: "task", width: size[0], height: size[1]}
			m.Update(tuiUpdate{})
			if m.mode != "preview" || !strings.Contains(m.View().Content, "Enter"+m.previewConfirm()) {
				t.Fatalf("%s preview context: %s", action, m.View().Content)
			}
			if action == "clean" && strings.Contains(m.View().Content, "确认同步") {
				t.Fatal("clean presented as sync")
			}
			m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
			m.beginEdit(6, "保留参数")
			before := m.View().Content
			for i := 0; i < 3; i++ {
				m.Update(tuiUpdate{})
			}
			if m.mode != "" || !m.editing || m.View().Content != before {
				t.Fatal("duplicate terminal event reopened dismissed preview or moved input")
			}
			// A new task completing while the user inspects history may update data,
			// but must not navigate away. This is a model boundary, not native timing.
			m.editing, m.mode = false, "history"
			c.tasks = append(c.tasks, UITask{ID: 2, Request: r, State: "succeeded"})
			m.Update(tuiUpdate{})
			if m.mode != "history" {
				t.Fatal("completion stole history focus")
			}
		}
	}
}

func TestUIExternalPlanBinding(t *testing.T) {
	c := NewUIController(t.TempDir())
	r := UIRequest{Directory: t.TempDir(), Action: "external", ID: "isolated-install", Manager: "manager", ExternalAction: "remove"}
	data, _ := json.Marshal(core.ExternalPlan{Manager: "manager", Args: []string{"remove", "exact-target"}})
	c.tasks = []UITask{{ID: 4, Request: r, State: "succeeded", Result: data}}
	for _, alter := range []func(*UIRequest){func(r *UIRequest) { r.ID = "different" }, func(r *UIRequest) { r.ExternalAction = "upgrade" }, func(r *UIRequest) { r.Global = true }, func(r *UIRequest) { r.Directory = t.TempDir() }, func(r *UIRequest) { r.Manager = "other" }, func(r *UIRequest) { r.PlanTaskID = 3 }} {
		bad := r
		bad.Apply = true
		bad.PlanTaskID = 4
		alter(&bad)
		if _, err := c.Start(bad); err == nil {
			t.Fatal("unverified external mutation accepted")
		}
	}
	good := r
	good.Apply = true
	good.PlanTaskID = 4
	if err := c.bindExternalPlan(&good); err != nil || good.externalPlan == nil || good.externalPlan.Args[1] != "exact-target" {
		t.Fatalf("snapshot lost: %v", err)
	}
	good.externalPlan.Args[1] = "modified"
	good.externalPlan = nil
	if err := c.bindExternalPlan(&good); err != nil || good.externalPlan.Args[1] != "exact-target" {
		t.Fatal("caller mutated retained plan")
	}
}

func TestUIEnvironmentSemantics(t *testing.T) {
	w := &core.Workbench{Desired: map[string]string{"java": "21"}, Applied: map[string]config.RuntimeLock{"java": {Version: "jdk-21.0.12.1+1"}}, Status: &core.Status{Environment: "ready"}}
	if uiToolState(w, "java") != "已应用" {
		t.Fatal("selector mistaken for drift")
	}
	w.Status.Environment = "drifted"
	if uiToolState(w, "java") != "已保存，待同步" {
		t.Fatal("digest drift hidden")
	}
	delete(w.Desired, "java")
	if uiToolState(w, "java") != "配置已移除，待同步" {
		t.Fatal("pending removal hidden")
	}
}
