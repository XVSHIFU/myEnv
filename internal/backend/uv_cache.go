package backend

import (
	"context"
	"fmt"
	"myenv/internal/runner"
	"os"
	"path/filepath"
)

// CleanCache delegates cache locking and ownership rules to pinned uv. Never
// pass --force: an in-use cache must retain the backend's normal protection.
// attempted is conservative on failure, since uv may already have removed files.
func (u UV) CleanCache(ctx context.Context, cache string) (attempted bool, err error) {
	if !filepath.IsAbs(cache) {
		return false, fmt.Errorf("uv cache directory must be absolute")
	}
	info, err := os.Lstat(cache)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("uv cache must be a directory, not a link or file")
	}
	if err = u.VerifyVersion(ctx); err != nil {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	output := &boundedOutput{limit: 8192}
	code, err := executeBackend(ctx, runner.Process{Executable: u.Executable, Args: []string{"cache", "clean", "--cache-dir", cache, "--no-config", "--offline", "--no-python-downloads", "--no-progress", "--color", "never"}, Directory: filepath.Dir(cache), Environment: uvEnvironment(os.Environ()), Stdout: output, Stderr: output})
	if err != nil {
		return true, err
	}
	if code != 0 {
		return true, fmt.Errorf("uv cache cleanup failed (%d): %s", code, output.data)
	}
	return true, nil
}
