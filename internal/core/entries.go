package core

import (
	"os"
	"path/filepath"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"myenv/internal/state"
)

// entriesHealthy follows Python venv links, but requires Node's regular entry.
func entriesHealthy(g *state.Generation, tools map[string]string) bool {
	for tool := range tools {
		var info os.FileInfo
		var err error
		switch tool {
		case "node":
			info, err = os.Lstat(g.NodeExecutable)
		case "python":
			info, err = os.Stat(g.PythonExecutable)
		case "java", "go", "rust":
			platform, e := runner.Platform()
			if e != nil {
				return false
			}
			if backend.CheckSDKEntries(g.Directory, tool, platform) != nil {
				return false
			}
			info, err = os.Lstat(backend.SDKEntry(g.Directory, tool, platform))
		default:
			return false
		}
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func lockHasTools(locked map[string]config.RuntimeLock, tools map[string]string) bool {
	if len(locked) != len(tools) {
		return false
	}
	for tool := range tools {
		if locked[tool].Version == "" {
			return false
		}
	}
	return true
}

func platformLockHealthy(locked config.PlatformLock, c *config.Config) bool {
	if !lockHasTools(locked.Tools, c.Tools) {
		return false
	}
	if c.Python == nil {
		return locked.Python == nil
	}
	return locked.Python != nil && locked.Python.Project == filepath.ToSlash(filepath.Clean(c.Python.Project))
}
