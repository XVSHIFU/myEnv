package cli

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"myenv/internal/backend"
)

func TestTUIJavaReadOnlyReleaseCannotSelect(t *testing.T) {
	c := NewUIController(t.TempDir())
	defer c.Close()
	request := UIRequest{Tool: "java", Action: "versions", Preview: true}
	m := &tuiModel{controller: c, width: 80, height: 24, request: request, catalogRequest: request, versionMode: true, catalog: &VersionResult{Releases: []backend.CatalogRelease{{Tool: "java", Version: "jdk-2026-09-14-13-26-beta", DisplayVersion: "Java 8 (1.8) · 8u502", Major: 8, Channel: "preview", Kind: "release_page", URL: "https://github.com/adoptium/temurin8-binaries/releases/tag/jdk-2026-09-14-13-26-beta"}}}}
	if !strings.Contains(m.View().Content, "只读发布记录") {
		t.Fatal("read-only release not labeled")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode == "save" || m.request.Selection != "" || c.Current().ID != 0 {
		t.Fatal("read-only release selected or started a task")
	}
}
