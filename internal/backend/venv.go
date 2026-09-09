package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"myenv/internal/runner"
)

func VerifyVenv(ctx context.Context, python, destination string) error {
	if !filepath.IsAbs(python) || !filepath.IsAbs(destination) {
		return fmt.Errorf("venv verification paths must be absolute")
	}
	output := &boundedOutput{limit: 32768}
	diagnostic := &boundedOutput{limit: 4096}
	code, err := executeBackend(ctx, runner.Process{Executable: python, Args: []string{"-I", "-c", "import json,sys; print(json.dumps(dict(prefix=sys.prefix,base_prefix=sys.base_prefix)))"}, Environment: uvEnvironment(os.Environ()), Stdout: output, Stderr: diagnostic})
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("venv verification failed (%d): %s", code, diagnostic.data)
	}
	var identity struct {
		Prefix     string `json:"prefix"`
		BasePrefix string `json:"base_prefix"`
	}
	if err = json.Unmarshal(output.data, &identity); err != nil {
		return fmt.Errorf("invalid venv identity: %w", err)
	}
	if !filepath.IsAbs(identity.Prefix) || !filepath.IsAbs(identity.BasePrefix) {
		return fmt.Errorf("invalid venv prefix paths")
	}
	prefix, err := os.Stat(identity.Prefix)
	if err != nil {
		return err
	}
	target, err := os.Stat(destination)
	if err != nil {
		return err
	}
	base, err := os.Stat(identity.BasePrefix)
	if err != nil {
		return err
	}
	if !prefix.IsDir() || !target.IsDir() || !os.SameFile(prefix, target) || os.SameFile(prefix, base) {
		return fmt.Errorf("Python does not belong to the selected virtual environment")
	}
	return nil
}
