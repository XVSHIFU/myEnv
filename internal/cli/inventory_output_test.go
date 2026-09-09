package cli

import (
	"bytes"
	"myenv/internal/core"
	"strings"
	"testing"
)

func TestInventoryOutputDetailsAndAttention(t *testing.T) {
	inventory := core.Inventory{Installations: []core.Installation{
		{Tool: "java", Owner: "myenv", Manager: "myenv", State: "available", Version: `openjdk version "17.0.20.1"`, ID: "internal-id", Path: `C:\中文目录\java.exe`},
		{Tool: "python", Owner: "external", Manager: "unknown", State: "alias"},
		{Tool: "go", Owner: "external", Manager: "unknown", State: "broken", Problem: "probe failed"},
	}}
	var out bytes.Buffer
	writeInventory(&out, inventory, false, true)
	if strings.Contains(out.String(), "internal-id") || strings.Contains(out.String(), "\t") || !strings.Contains(out.String(), "java 17.0.20.1") || !strings.Contains(out.String(), "probe failed") || !strings.Contains(out.String(), "应用执行别名") {
		t.Fatal(out.String())
	}
	out.Reset()
	writeInventory(&out, inventory, true, true)
	if !strings.Contains(out.String(), inventory.Installations[0].Path) || !strings.Contains(out.String(), "internal-id") {
		t.Fatal(out.String())
	}
}

func TestRunMissingCommandHasExecutableExample(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := execute([]string{"--lang", "zh-CN", "run", "--global"}, strings.NewReader(""), &out, &diagnostic, "test", t.TempDir())
	if code == 0 || !strings.Contains(diagnostic.String(), "myenv run --global java -version") || strings.Contains(diagnostic.String(), "requires at least") {
		t.Fatalf("%d: %s", code, &diagnostic)
	}
}
