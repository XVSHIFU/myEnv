package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/state"
)

// managedUV runs under the workspace modification lock. Interrupted installs
// have no published entry and are never reused as a complete backend.
func (s *Service) managedUV(ctx context.Context, work, platform string) (backend.UV, error) {
	if s.UV != nil {
		return *s.UV, s.UV.VerifyVersion(ctx)
	}
	storage, err := s.storagePaths(work)
	if err != nil {
		return backend.UV{}, err
	}
	directory := filepath.Join(storage.Data, "backends")
	if err = os.MkdirAll(directory, 0700); err != nil {
		return backend.UV{}, err
	}
	unlock, err := state.LockWorkspace(ctx, filepath.Join(directory, "uv-"+backend.UVVersion+"-"+platform+".lock"))
	if err != nil {
		return backend.UV{}, err
	}
	defer unlock()
	pointer := filepath.Join(directory, "uv-"+backend.UVVersion+"-"+platform)
	if data, err := config.ReadInput(pointer); err == nil {
		entry, err := config.Within(directory, strings.TrimSpace(string(data)))
		if err != nil {
			return backend.UV{}, err
		}
		u := backend.UV{Executable: entry}
		return u, u.VerifyVersion(ctx)
	} else if !os.IsNotExist(err) {
		return backend.UV{}, err
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return backend.UV{}, err
	}
	preparation, err := os.MkdirTemp(directory, "uv-prepare-")
	if err != nil {
		return backend.UV{}, err
	}
	archive := s.UVArchive
	if archive == "" {
		_, archive, err = backend.DownloadUVFromMirror(ctx, platform, preparation, s.UVMirror)
		if err != nil {
			return backend.UV{}, err
		}
		defer os.Remove(archive)
	}
	u, err := backend.PrepareUV(ctx, archive, filepath.Join(preparation, "payload"), platform)
	if err != nil {
		return backend.UV{}, err
	}
	relative, err := filepath.Rel(directory, u.Executable)
	if err != nil {
		return backend.UV{}, err
	}
	f, err := os.CreateTemp(directory, ".uv-entry-")
	if err != nil {
		return backend.UV{}, err
	}
	defer os.Remove(f.Name())
	_, err = f.WriteString(relative + "\n")
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return backend.UV{}, err
	}
	if closeErr != nil {
		return backend.UV{}, closeErr
	}
	return u, os.Rename(f.Name(), pointer)
}

func pythonInputs(root string, c *config.Config, locked bool) (*config.PythonInputs, error) {
	if c.Python == nil {
		return nil, nil
	}
	inputs, err := config.ReadPythonInputs(root, c.Python.Project, locked)
	if err != nil {
		return nil, err
	}
	return &inputs, nil
}

func pythonEntry(venv, platform string) string {
	if platform == "windows-amd64" {
		return filepath.Join(venv, "Scripts", "python.exe")
	}
	return filepath.Join(venv, "bin", "python")
}

func samePythonManifest(a, b *config.PythonInputs) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Project == b.Project && a.PyprojectSHA256 == b.PyprojectSHA256 && a.UVConfigSHA256 == b.UVConfigSHA256 && a.RelatedSHA256 == b.RelatedSHA256 && a.WorkspaceRoot == b.WorkspaceRoot && a.WorkspaceSHA256 == b.WorkspaceSHA256
}

func (s *Service) preparePython(ctx context.Context, u backend.UV, version, work, directory, platform string) (string, string, error) {
	storage, err := s.storagePaths(work)
	if err != nil {
		return "", "", err
	}
	python, err := s.installSharedPython(ctx, u, version, platform, storage)
	if err != nil {
		return "", "", err
	}
	venv := filepath.Join(directory, "venv")
	if err = os.MkdirAll(directory, 0700); err != nil {
		return "", "", err
	}
	if err = u.CreateVenv(ctx, python, venv, filepath.Join(storage.Cache, "uv")); err != nil {
		return "", "", fmt.Errorf("prepare Python venv: %w", err)
	}
	return python, pythonEntry(venv, platform), nil
}

// Only the shared interpreter is covered by this lock. Project-local venv
// creation and dependency synchronization continue under their project locks.
func (s *Service) installSharedPython(ctx context.Context, u backend.UV, version, platform string, storage config.UserStorage) (string, error) {
	if s.Storage != nil {
		locks := filepath.Join(storage.Data, "locks")
		if err := os.MkdirAll(locks, 0700); err != nil {
			return "", err
		}
		unlock, err := state.LockWorkspace(ctx, filepath.Join(locks, "python-"+platform+"-"+version+".lock"))
		if err != nil {
			return "", err
		}
		defer unlock()
	}
	runtimes := s.PythonDirectory
	if runtimes == "" {
		runtimes = filepath.Join(storage.Data, "runtimes")
	}
	return u.InstallPythonFromMirror(ctx, version, runtimes, filepath.Join(storage.Cache, "uv"), s.PythonMirror)
}
