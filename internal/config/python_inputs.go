package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PythonInputs struct {
	ConfigPolicy    string `json:"config_policy,omitempty"`
	Project         string `json:"project"`
	PyprojectSHA256 string `json:"pyproject_sha256"`
	UVLockSHA256    string `json:"uv_lock_sha256,omitempty"`
	UVConfigSHA256  string `json:"uv_config_sha256,omitempty"`
	RelatedSHA256   string `json:"related_sha256,omitempty"`
	WorkspaceRoot   string `json:"workspace_root,omitempty"`
	WorkspaceSHA256 string `json:"workspace_sha256,omitempty"`
}

type pythonInputRead struct {
	data []byte
	err  error
}

// ReadPythonInputs bounds reads and records missing optional files distinctly
// from empty files. Call again before publication to detect external edits.
func ReadPythonInputs(workspace, project string, locked bool) (PythonInputs, error) {
	var result PythonInputs
	// At most the project/workspace manifests and root uv.toml are retained.
	// This snapshot is local to one check, never shared across commands.
	metadata := make(map[string]pythonInputRead, 3)
	directory, err := Within(workspace, project)
	if err != nil {
		return result, err
	}
	result.Project = filepath.ToSlash(filepath.Clean(project))
	result.ConfigPolicy = "project-only-v1"
	lockRoot, err := PythonWorkspaceRoot(workspace, project)
	if err != nil {
		return result, err
	}
	if filepath.Clean(lockRoot) != filepath.Clean(project) {
		result.WorkspaceRoot = filepath.ToSlash(lockRoot)
		data, err := ReadInput(filepath.Join(workspace, lockRoot, "pyproject.toml"))
		if err != nil {
			return result, err
		}
		metadata[filepath.Clean(filepath.Join(lockRoot, "pyproject.toml"))] = pythonInputRead{data: data}
		hash := sha256.Sum256(data)
		result.WorkspaceSHA256 = hex.EncodeToString(hash[:])
	}
	for _, input := range []struct {
		name     string
		required bool
		digest   *string
	}{
		{"pyproject.toml", true, &result.PyprojectSHA256},
		{"uv.lock", locked, &result.UVLockSHA256},
		{"uv.toml", false, &result.UVConfigSHA256},
	} {
		inputProject := lockRoot
		if input.name == "pyproject.toml" {
			inputProject = project
		}
		path, err := Within(workspace, filepath.Join(inputProject, input.name))
		if err != nil {
			return result, err
		}
		data, err := ReadInput(path)
		if input.name != "uv.lock" {
			metadata[filepath.Clean(filepath.Join(inputProject, input.name))] = pythonInputRead{data: data, err: err}
		}
		if os.IsNotExist(err) && !input.required {
			continue
		}
		if err != nil {
			if input.name == "uv.lock" && os.IsNotExist(err) {
				return result, fmt.Errorf("LOCK_OUT_OF_DATE: %s requires %s", directory, filepath.Join(workspace, lockRoot, "uv.lock"))
			}
			return result, err
		}
		hash := sha256.Sum256(data)
		*input.digest = hex.EncodeToString(hash[:])
	}
	result.RelatedSHA256, err = pythonRelatedDigest(workspace, lockRoot, metadata)
	return result, err
}

func (p PythonInputs) Digest() string {
	data, _ := json.Marshal(p)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
