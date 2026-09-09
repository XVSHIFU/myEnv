package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"myenv/internal/config"
	"myenv/internal/runner"
)

// InstallPython delegates platform patching to fixed uv. Its result provides
// version evidence, not a claim that the Python archive has an artifact lock.
func (u UV) InstallPython(ctx context.Context, version, directory, cache string) (string, error) {
	return u.InstallPythonFromMirror(ctx, version, directory, cache, "")
}

func (u UV) InstallPythonFromMirror(ctx context.Context, version, directory, cache, mirror string) (string, error) {
	base, err := config.MirrorBaseURL(mirror)
	if err != nil {
		return "", err
	}
	if _, err := exactNodeVersion(version); err != nil && !config.IsPreview("python", version) {
		return "", fmt.Errorf("Python installation requires an exact three-part version")
	}
	if !filepath.IsAbs(directory) || !filepath.IsAbs(cache) {
		return "", fmt.Errorf("Python install and cache directories must be absolute")
	}
	if err := u.VerifyVersion(ctx); err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	env := append(uvEnvironment(os.Environ()), "UV_PYTHON_INSTALL_DIR="+directory)
	output := &boundedOutput{limit: 4096}
	args := []string{"python", "install", version, "--install-dir", directory, "--no-bin", "--no-registry", "--no-config", "--no-progress", "--cache-dir", cache}
	if base != "" {
		args = append(args, "--mirror", base)
	}
	code, err := executeBackend(ctx, runner.Process{Executable: u.Executable, Args: args, Directory: directory, Environment: env, Stdout: output, Stderr: output})
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("uv Python install failed (%d): %s", code, output.data)
	}
	stdout := &boundedOutput{limit: 32768}
	stderr := &boundedOutput{limit: 4096}
	code, err = executeBackend(ctx, runner.Process{Executable: u.Executable, Args: []string{"python", "find", version, "--managed-python", "--no-python-downloads", "--no-project", "--no-config", "--offline", "--cache-dir", cache}, Directory: directory, Environment: env, Stdout: stdout, Stderr: stderr})
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("uv managed Python lookup failed (%d): %s", code, stderr.data)
	}
	python := strings.TrimSpace(string(stdout.data))
	if !filepath.IsAbs(python) {
		return "", fmt.Errorf("uv returned a non-absolute interpreter")
	}
	relative, err := filepath.Rel(directory, python)
	if err != nil {
		return "", err
	}
	python, err = config.Within(directory, relative)
	if err != nil {
		return "", fmt.Errorf("uv interpreter escaped managed directory: %w", err)
	}
	if err = verifyPython(ctx, python, version); err != nil {
		return "", err
	}
	return python, nil
}

func verifyPython(ctx context.Context, python, version string) error {
	output := &boundedOutput{limit: 4096}
	code, err := executeBackend(ctx, runner.Process{Executable: python, Args: []string{"-I", "-c", "import platform; print(platform.python_version())"}, Environment: uvEnvironment(os.Environ()), Stdout: output, Stderr: output})
	if err != nil {
		return err
	}
	if code != 0 || strings.TrimSpace(string(output.data)) != version {
		return fmt.Errorf("managed Python version differs from selected version")
	}
	return nil
}
