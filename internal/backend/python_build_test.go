package backend

import (
	"errors"
	"github.com/pelletier/go-toml/v2"
	"myenv/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestFirstPartyBuildLocations(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"unrelated", "[tool.docs]\npath='docs'\ndynamic=true\npackage=true", false},
		{"wheel-registry", "[project]\ndependencies=['requests>=2']", false},
		{"backend", "[build-system]\nrequires=[]", true},
		{"dynamic", "[project]\ndynamic=['version']", true},
		{"empty-dynamic", "[project]\ndynamic=[]", false},
		{"local", "[tool.uv.sources]\nexample={path='../example'}", true},
		{"registry", "[tool.uv.sources]\nexample={index='private'}", false},
		{"direct-reference", "[project]\ndependencies=['example @ file:///tmp/example']", true},
		{"group-reference", "[dependency-groups]\ndev=['example@git+https://example.test/repo']", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var document map[string]any
			if err := toml.Unmarshal([]byte(tc.input), &document); err != nil {
				t.Fatal(err)
			}
			if got := requiresFirstPartyBuild(document); got != tc.want {
				t.Fatalf("build requirement %t, want %t", got, tc.want)
			}
		})
	}
}

func TestParentWorkspaceBuildPreflight(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "packages", "example")
	if err := os.MkdirAll(project, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "pyproject.toml"), []byte("[project]\nname='example'\nversion='1'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.uv.workspace]\nmembers=['packages/*']\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := checkFirstPartyBuild(project)
	var needs *config.NeedsInput
	if !errors.As(err, &needs) {
		t.Fatalf("parent workspace bypassed build permission: %v", err)
	}
}
