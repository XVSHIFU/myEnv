package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"myenv/internal/runner"
)

// UVVersion is deliberately fixed; upgrades require a backend release change.
const UVVersion = "0.11.26"

type UV struct{ Executable string }

// VerifyVersion checks an explicitly supplied managed engine entry. Distribution
// hash verification belongs to the installer before this executable is used.
func (u UV) VerifyVersion(ctx context.Context) error {
	if !filepath.IsAbs(u.Executable) {
		return fmt.Errorf("uv entry must be absolute")
	}
	info, err := os.Lstat(u.Executable)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("uv entry must be a regular file")
	}
	output := &boundedOutput{limit: 4096}
	code, err := executeBackend(ctx, runner.Process{Executable: u.Executable, Args: []string{"--version"}, Environment: uvEnvironment(os.Environ()), Stdout: output, Stderr: output})
	if err != nil {
		return err
	}
	fields := strings.Fields(string(output.data))
	if code != 0 || len(fields) < 2 || fields[0] != "uv" || fields[1] != UVVersion {
		return fmt.Errorf("uv backend requires version %s", UVVersion)
	}
	return nil
}

// CreateVenv uses an already selected absolute interpreter and a new final path.
// It does not download Python, seed dependencies, or relocate an environment.
func (u UV) CreateVenv(ctx context.Context, python, destination, cache string) error {
	for _, p := range []string{python, destination, cache} {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("uv preparation paths must be absolute")
		}
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("venv destination must not exist")
	}
	if err := u.VerifyVersion(ctx); err != nil {
		return err
	}
	output := &boundedOutput{limit: 4096}
	code, err := executeBackend(ctx, runner.Process{Executable: u.Executable, Args: []string{"venv", "--python", python, "--no-python-downloads", "--no-project", "--no-config", "--offline", "--cache-dir", cache, destination}, Directory: filepath.Dir(destination), Environment: uvEnvironment(os.Environ()), Stdout: output, Stderr: output})
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("uv venv failed (%d): %s", code, output.data)
	}
	entry := filepath.Join(destination, "bin", "python")
	if runtime.GOOS == "windows" {
		entry = filepath.Join(destination, "Scripts", "python.exe")
	}
	return VerifyVenv(ctx, entry, destination)
}

func uvEnvironment(parent []string) []string {
	result := make([]string, 0, len(parent))
	for _, entry := range parent {
		key, _, ok := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if ok && (strings.HasPrefix(upper, "UV_") || upper == "PYTHONHOME" || upper == "PYTHONPATH" || upper == "VIRTUAL_ENV" || upper == "CONDA_PREFIX") {
			continue
		}
		result = append(result, entry)
	}
	return result
}
