package runner

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvironmentProjectCaseConflict(t *testing.T) {
	project := map[string]string{"Path": "first-private-value", "PATH": "second-private-value"}
	if _, err := Environment(nil, project, nil, true); err == nil || err.Error() != `duplicate case-insensitive environment key "PATH"` {
		t.Fatalf("Windows conflict diagnostic: %v", err)
	}
	env, err := Environment(nil, project, nil, false)
	if err != nil || len(env) != 2 || env[0] != "PATH=second-private-value" || env[1] != "Path=first-private-value" {
		t.Fatalf("case-sensitive environment was changed: %v %v", env, err)
	}
}

func TestEnvironmentIsolation(t *testing.T) {
	parent := []string{"Path=old", "PATH=last", "TOKEN=secret", "=C:=C:\\work"}
	bin := filepath.Join(t.TempDir(), "node")
	env, err := Environment(parent, map[string]string{"APP_ENV": "test", "path": "project"}, []string{bin}, true)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "PATH="+bin+";project") || !strings.Contains(joined, "TOKEN=secret") || !strings.Contains(joined, "=C:=C:\\work") {
		t.Fatalf("wrong child environment: %v", env)
	}
	count := 0
	for _, v := range env {
		if strings.HasPrefix(strings.ToUpper(v), "PATH=") {
			count++
		}
	}
	if count != 1 {
		t.Fatal("duplicate Windows PATH")
	}
	if parent[0] != "Path=old" {
		t.Fatal("mutated parent")
	}
	if _, err = Environment(parent, map[string]string{"BAD=KEY": "x"}, nil, true); err == nil {
		t.Fatal("accepted invalid environment key")
	}
}
