package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExternalCondaMembershipAndBaseProtection(t *testing.T) {
	root := t.TempDir()
	prefix := filepath.Join(root, "env")
	os.MkdirAll(filepath.Join(prefix, "conda-meta"), 0700)
	os.WriteFile(filepath.Join(prefix, "conda-meta", "history"), nil, 0600)
	path := filepath.Join(prefix, "python.exe")
	manager := filepath.Join(root, "conda.exe")
	if runtime.GOOS != "windows" {
		path = filepath.Join(prefix, "bin", "python")
		manager = filepath.Join(root, "conda")
	}
	os.WriteFile(manager, nil, 0700)
	row := Installation{Owner: "external", Tool: "python", Path: path}
	base := filepath.Join(root, "base")
	member := true
	read := func(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
		if len(args) == 5 && args[0] == "list" && args[2] == prefix {
			return []byte(`[{"name":"python","version":"3.12.13","build":"h123_0"}]`), nil
		}
		switch strings.Join(args, " ") {
		case "--version":
			return []byte("conda 26.3.1"), nil
		case "info --json":
			return json.Marshal(map[string]string{"root_prefix": base})
		case "env list --json":
			if member {
				return json.Marshal(map[string]any{"envs": []string{prefix}})
			}
			return []byte(`{"envs":[]}`), nil
		}
		return nil, fmt.Errorf("unexpected query %v", args)
	}
	plan, e := buildExternalPlan(context.Background(), row, manager, "remove", read)
	if e != nil || strings.Join(plan.Args, " ") != "env remove --prefix "+prefix+" --yes" {
		t.Fatal(plan, e)
	}
	plan, e = buildExternalPlan(context.Background(), row, manager, "repair", read)
	if e != nil || !strings.Contains(strings.Join(plan.Args, " "), "python=3.12.13=h123_0") {
		t.Fatal("repair must pin installed version/build", plan, e)
	}
	member = false
	if _, e = buildExternalPlan(context.Background(), row, manager, "remove", read); e == nil {
		t.Fatal("unregistered prefix accepted")
	}
	member = true
	base = prefix
	if _, e = buildExternalPlan(context.Background(), row, manager, "repair", read); e == nil {
		t.Fatal("base environment accepted")
	}
}

func TestExternalCannotBypassMyenvProtection(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".myenv", "runtimes", "python.exe")
	if _, e := BuildExternalPlan(context.Background(), Installation{Owner: "external", Tool: "python", Path: path}, "ignored", "remove"); e == nil || !strings.Contains(e.Error(), "myenv generation") {
		t.Fatal(e)
	}
}
