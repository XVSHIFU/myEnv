package backend

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"myenv/internal/config"
)

// projectUVConfig makes configuration discovery explicit. uv.toml has the
// native precedence over tool.uv; absent settings yield an empty config rather
// than inheritance from an unrelated parent, user, or system directory.
func projectUVConfig(project string) (string, error) {
	data, err := config.ReadInput(filepath.Join(project, "uv.toml"))
	if os.IsNotExist(err) {
		data, err = config.ReadInput(filepath.Join(project, "pyproject.toml"))
		if err != nil {
			return "", err
		}
		var document map[string]any
		if err = toml.Unmarshal(data, &document); err != nil {
			return "", err
		}
		tool, _ := document["tool"].(map[string]any)
		uv, _ := tool["uv"].(map[string]any)
		if uv == nil {
			uv = map[string]any{}
		}
		// Pinned uv-settings::validate_uv_toml rejects these fields in a
		// standalone config. uv reads them separately from the project itself.
		for _, key := range []string{"conflicts", "workspace", "sources", "dev-dependencies", "default-groups", "dependency-groups", "managed", "package", "build-backend"} {
			delete(uv, key)
		}
		data, err = toml.Marshal(uv)
	}
	if err != nil {
		return "", err
	}
	// Keep the native configuration origin for settings containing relative
	// paths. This invocation-only file is removed by the caller.
	f, err := os.CreateTemp(project, ".myenv-uv-config-*.toml")
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		os.Remove(f.Name())
		if err != nil {
			return "", err
		}
		return "", closeErr
	}
	return f.Name(), nil
}
