package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type RuntimeLock struct {
	RuntimeVersion string `json:"runtime_version,omitempty"`
	Version        string `json:"version"`
	Backend        string `json:"backend"`
	Evidence       string `json:"evidence"`
	URL            string `json:"url,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	NPM            string `json:"npm,omitempty"`
}

var exactLockedVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// ReplaceLock requires the caller to hold the workspace modification lock.
// Desired lock publication is separate from the SQLite active-generation commit.
func ReplaceLock(path string, lock *Lock) error {
	if err := lock.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".myenv-lock-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	_, err = file.Write(append(data, '\n'))
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return replaceFile(temporary, path)
}

type PlatformLock struct {
	Tools  map[string]RuntimeLock `json:"tools"`
	Python *PythonInputs          `json:"python,omitempty"`
}
type Lock struct {
	Schema       int                     `json:"schema"`
	ConfigDigest string                  `json:"config_digest"`
	Platforms    map[string]PlatformLock `json:"platforms"`
}

// Digest describes semantic configuration, independent of YAML comments/order.
func Digest(c *Config) (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func ReadLock(path string) (*Lock, error) {
	data, err := ReadInput(path)
	if err != nil {
		return nil, err
	}
	if err = checkJSONKeys(data); err != nil {
		return nil, fmt.Errorf("invalid myenv.lock: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var lock Lock
	if err = decoder.Decode(&lock); err != nil {
		return nil, fmt.Errorf("invalid myenv.lock: %w", err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("myenv.lock must contain one JSON object")
	}
	if err = lock.Validate(); err != nil {
		return nil, err
	}
	return &lock, nil
}
func (l *Lock) Validate() error {
	digest, err := hex.DecodeString(l.ConfigDigest)
	if l.Schema != 1 || err != nil || len(digest) != 32 || len(l.Platforms) == 0 {
		return fmt.Errorf("invalid lock schema, config digest or platforms")
	}
	for platform, record := range l.Platforms {
		if platform == "" || record.Tools == nil {
			return fmt.Errorf("empty platform lock")
		}
		if record.Python != nil {
			p := record.Python
			if p.ConfigPolicy != "" && p.ConfigPolicy != "project-only-v1" {
				return fmt.Errorf("unsupported Python config policy")
			}
			if record.Tools["python"].Version == "" || p.Project == "" || filepath.IsAbs(p.Project) || p.Project == ".." || strings.HasPrefix(filepath.ToSlash(p.Project), "../") {
				return fmt.Errorf("invalid Python project lock")
			}
			if (p.WorkspaceRoot == "") != (p.WorkspaceSHA256 == "") {
				return fmt.Errorf("Python workspace root and digest must be recorded together")
			}
			if p.WorkspaceRoot != "" && (filepath.IsAbs(p.WorkspaceRoot) || filepath.VolumeName(p.WorkspaceRoot) != "" || p.WorkspaceRoot == ".." || strings.HasPrefix(filepath.ToSlash(p.WorkspaceRoot), "../")) {
				return fmt.Errorf("invalid Python workspace root")
			}
			for index, value := range []string{p.PyprojectSHA256, p.UVLockSHA256, p.UVConfigSHA256, p.RelatedSHA256, p.WorkspaceSHA256} {
				if index >= 2 && value == "" {
					continue
				}
				decoded, err := hex.DecodeString(value)
				if err != nil || len(decoded) != 32 {
					return fmt.Errorf("invalid Python input digest")
				}
			}
		}
		for tool, runtime := range record.Tools {
			constraint, err := ParseConstraint(tool, runtime.Version)
			if err != nil || (!(tool == "java" || tool == "go" || tool == "rust" || (tool == "python" && pythonPreview.MatchString(runtime.Version)) || (tool == "node" && NodeChannel(runtime.Version) != "")) && !exactLockedVersion.MatchString(runtime.Version)) || !constraint.Contains(runtime.Version) || runtime.Backend == "" {
				return fmt.Errorf("invalid locked %s runtime", tool)
			}
			if runtime.Evidence != "version" && runtime.Evidence != "artifact" {
				return fmt.Errorf("invalid locking evidence")
			}
			if runtime.Evidence == "artifact" {
				digest, err := hex.DecodeString(runtime.SHA256)
				if err != nil || len(digest) != 32 || runtime.URL == "" {
					return fmt.Errorf("artifact lock requires URL and SHA256")
				}
			}
		}
	}
	return nil
}

// WriteLock writes only a new lock. Replacement must be coordinated by core's
// workspace lock and durable publication policy; do not silently truncate a lock.
func WriteNewLock(path string, lock *Lock) error {
	if err := lock.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
