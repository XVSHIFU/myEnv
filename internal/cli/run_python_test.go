package cli

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"myenv/internal/runner"
)

func TestPythonRunEnvironment(t *testing.T) {
	for _, windows := range []bool{false, true} {
		name := "unix"
		if windows {
			name = "windows"
		}
		t.Run(name, func(t *testing.T) {
			venv := filepath.Join(t.TempDir(), "venv")
			bin := filepath.Join(venv, "bin")
			if windows {
				bin = filepath.Join(venv, "Scripts")
			}
			parent := []string{"PYTHONHOME=other-python", "VIRTUAL_ENV=other-venv", "PYTHONPATH=custom-modules", "CONDA_PREFIX=external-conda", "HTTPS_PROXY=proxy", "pythonhome=case-sensitive-value"}
			original := append([]string(nil), parent...)
			overrides := map[string]string{"APP_MODE": "test"}
			filtered := pythonRunEnvironment(parent, overrides, filepath.Join(bin, "python"), windows)
			// Exercise variable case rules independently of host-specific PATH syntax.
			// A Windows drive colon is not a valid Unix PATH component.
			env, err := runner.Environment(filtered, overrides, nil, windows)
			if err != nil {
				t.Fatal(err)
			}
			values := make(map[string]string)
			for _, entry := range env {
				key, value, _ := strings.Cut(entry, "=")
				if windows {
					key = strings.ToUpper(key)
				}
				values[key] = value
			}
			if _, exists := values["PYTHONHOME"]; exists {
				t.Fatal("inherited PYTHONHOME overrides the selected venv")
			}
			if values["VIRTUAL_ENV"] != venv || values["PYTHONPATH"] != "custom-modules" || values["CONDA_PREFIX"] != "external-conda" || values["HTTPS_PROXY"] != "proxy" || values["APP_MODE"] != "test" {
				t.Fatalf("wrong managed environment: %v", env)
			}
			if !windows && values["pythonhome"] != "case-sensitive-value" {
				t.Fatal("removed unrelated case-sensitive variable")
			}
			if !reflect.DeepEqual(parent, original) {
				t.Fatal("modified parent environment")
			}

			// Explicit configuration is still authoritative, including Windows
			// case variants; adding a default must not introduce duplicate keys.
			key := "VIRTUAL_ENV"
			if windows {
				key = "Virtual_Env"
			}
			overrides = map[string]string{key: "explicit-venv", "PYTHONHOME": "explicit-home"}
			filtered = pythonRunEnvironment(parent, overrides, filepath.Join(bin, "python"), windows)
			env, err = runner.Environment(filtered, overrides, nil, windows)
			if err != nil || !strings.Contains(strings.Join(env, "\n"), key+"=explicit-venv") || !strings.Contains(strings.Join(env, "\n"), "PYTHONHOME=explicit-home") {
				t.Fatalf("explicit configuration changed: %v %v", env, err)
			}
		})
	}
}

func TestPythonRunEnvironmentWithoutManagedPython(t *testing.T) {
	parent := []string{"VIRTUAL_ENV=external", "PYTHONHOME=external", "PYTHONPATH=external"}
	overrides := map[string]string{}
	got := pythonRunEnvironment(parent, overrides, "", false)
	if !reflect.DeepEqual(got, parent) || len(overrides) != 0 {
		t.Fatal("changed Python environment for a project without managed Python")
	}
}
