package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"myenv/internal/config"
	"myenv/internal/runner"
)

type PythonProjectRequest struct {
	Project, Python, Venv, Cache string
	ConfigProject                string
	Groups                       []string
	Locked, AllowBuild, Offline  bool
}

// SyncPythonProject updates/checks the native lock, then synchronizes only the
// explicitly selected venv. Publication remains the caller's responsibility.
func (u UV) SyncPythonProject(ctx context.Context, r PythonProjectRequest) error {
	for _, p := range []string{r.Project, r.Python, r.Venv, r.Cache} {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("Python project paths must be absolute")
		}
	}
	if !r.AllowBuild {
		if err := checkFirstPartyBuild(r.Project); err != nil {
			return err
		}
	}
	if err := u.VerifyVersion(ctx); err != nil {
		return err
	}
	entry := filepath.Join(r.Venv, "bin", "python")
	if runtime.GOOS == "windows" {
		entry = filepath.Join(r.Venv, "Scripts", "python.exe")
	}
	if err := VerifyVenv(ctx, entry, r.Venv); err != nil {
		return err
	}
	configProject := r.ConfigProject
	if configProject == "" {
		configProject = r.Project
	}
	if !filepath.IsAbs(configProject) {
		return fmt.Errorf("Python configuration project must be absolute")
	}
	configuration, err := projectUVConfig(configProject)
	if err != nil {
		return err
	}
	defer os.Remove(configuration)
	env := append(uvEnvironment(os.Environ()), "UV_PROJECT_ENVIRONMENT="+r.Venv)
	common := []string{"--python", r.Python, "--no-python-downloads", "--cache-dir", r.Cache, "--no-progress", "--color", "never", "--config-file", configuration}
	if !r.AllowBuild {
		common = append(common, "--no-build")
	}
	if r.Offline {
		common = append(common, "--offline")
	}
	lockArgs := append([]string{"lock"}, common...)
	if r.Locked {
		lockArgs = append(lockArgs, "--check")
	}
	syncArgs := append([]string{"sync", "--locked", "--no-default-groups", "--link-mode", "copy"}, common...)
	for _, group := range r.Groups {
		syncArgs = append(syncArgs, "--group", group)
	}
	for _, args := range [][]string{lockArgs, syncArgs} {
		output := &boundedOutput{limit: 8192}
		code, err := executeBackend(ctx, runner.Process{Executable: u.Executable, Args: args, Directory: r.Project, Environment: env, Stdout: output, Stderr: output})
		if err != nil {
			return err
		}
		if code != 0 {
			if !r.AllowBuild && sourceBuildDisabled(string(output.data)) {
				return &config.NeedsInput{Message: fmt.Sprintf("uv %s requires source build permission; retry with --allow-build.\n%s", args[0], output.data)}
			}
			return fmt.Errorf("uv %s failed (%d): %s", args[0], code, output.data)
		}
	}
	return nil
}

// Match the explicit --no-build diagnostic from the pinned uv release, not a
// general resolution/build failure, which can have unrelated causes.
func sourceBuildDisabled(output string) bool {
	return strings.Contains(output, "because building from source is disabled for all packages (i.e., with `--no-build`)") ||
		strings.Contains(output, "can't be installed because it is marked as `--no-build` but has no binary distribution")
}

// --no-build does not prohibit all first-party metadata hooks. Fail before uv
// for project/workspace/local source forms requiring explicit build permission.
func checkFirstPartyBuild(project string) error {
	if err := checkParentWorkspaces(project); err != nil {
		return err
	}
	for _, name := range []string{"pyproject.toml", "uv.toml"} {
		data, err := config.ReadInput(filepath.Join(project, name))
		if os.IsNotExist(err) && name == "uv.toml" {
			continue
		}
		if err != nil {
			return err
		}
		var document map[string]any
		if err = toml.Unmarshal(data, &document); err != nil {
			return err
		}
		if name == "uv.toml" {
			document = map[string]any{"tool": map[string]any{"uv": document}}
		}
		if requiresFirstPartyBuild(document) {
			return &config.NeedsInput{Message: "Python project or local workspace metadata may execute build code; retry with --allow-build."}
		}
	}
	return nil
}

// CheckPythonBuildPermission inspects project metadata without running uv or
// build hooks. The caller may obtain one invocation-scoped approval and retry.
func CheckPythonBuildPermission(project string) error { return checkFirstPartyBuild(project) }

func checkParentWorkspaces(project string) error {
	root, err := config.PythonNativeWorkspaceRoot(project)
	if err != nil {
		return err
	}
	if filepath.Clean(root) != filepath.Clean(project) {
		return &config.NeedsInput{Message: "A parent Python workspace may introduce build metadata; retry with --allow-build after reviewing its members."}
	}
	return nil
}

func requiresFirstPartyBuild(document map[string]any) bool {
	if _, ok := document["build-system"]; ok {
		return true
	}
	project, _ := document["project"].(map[string]any)
	if dynamic, ok := project["dynamic"].([]any); ok && len(dynamic) > 0 {
		return true
	}
	tool, _ := document["tool"].(map[string]any)
	uv, _ := tool["uv"].(map[string]any)
	if uv["package"] == true {
		return true
	}
	if _, ok := uv["workspace"]; ok {
		return true
	}
	if localSource(uv["sources"]) {
		return true
	}
	for _, requirements := range []any{project["dependencies"], project["optional-dependencies"], document["dependency-groups"], uv["dev-dependencies"]} {
		if directRequirements(requirements) {
			return true
		}
	}
	return false
}

func localSource(value any) bool {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			if key == "workspace" || key == "path" || key == "git" || key == "editable" {
				return true
			}
			if localSource(child) {
				return true
			}
		}
	case []any:
		for _, child := range node {
			if localSource(child) {
				return true
			}
		}
	}
	return false
}

func directRequirements(value any) bool {
	switch node := value.(type) {
	case string:
		return strings.Contains(node, "@")
	case []any:
		for _, child := range node {
			if directRequirements(child) {
				return true
			}
		}
	case map[string]any:
		for _, child := range node {
			if directRequirements(child) {
				return true
			}
		}
	}
	return false
}
