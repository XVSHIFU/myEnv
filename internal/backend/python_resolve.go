package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"myenv/internal/config"
	"myenv/internal/runner"
)

type PythonRelease struct {
	Key      string `json:"key"`
	Version  string `json:"version"`
	URL      string `json:"url"`
	Platform string `json:"platform"`
	Backend  string `json:"backend"`
	Evidence string `json:"evidence"`
}

func (u UV) ResolvePython(ctx context.Context, selector, cache string) (PythonRelease, error) {
	var result PythonRelease
	constraint, err := config.ParseConstraint("python", selector)
	if err != nil {
		return result, err
	}
	entries, err := u.pythonEntries(ctx, cache)
	if err != nil {
		return result, err
	}
	platform, err := runner.Platform()
	if err != nil {
		return result, err
	}
	best := [3]int{-1, -1, -1}
	for _, entry := range entries {
		if !constraint.Contains(entry.Version) {
			continue
		}
		if _, err := exactNodeVersion(entry.Version); err != nil && !(config.IsPreview("python", selector) && config.IsPreview("python", entry.Version)) {
			continue
		}
		parts := [3]int{entry.Parts.Major, entry.Parts.Minor, entry.Parts.Patch}
		newer := false
		for i := 0; i < 3; i++ {
			if parts[i] != best[i] {
				newer = parts[i] > best[i]
				break
			}
		}
		if newer {
			best = parts
			result = PythonRelease{Key: entry.Key, Version: entry.Version, URL: entry.URL, Platform: platform, Backend: "uv-" + UVVersion, Evidence: "version"}
		}
	}
	if result.Version == "" {
		return result, fmt.Errorf("no managed CPython matches %q in uv %s", selector, UVVersion)
	}
	return result, nil
}

type pythonCatalogEntry struct {
	Key, Version, URL, Implementation, Variant, Arch string
	Parts                                            struct{ Major, Minor, Patch int } `json:"version_parts"`
}

func (u UV) pythonEntries(ctx context.Context, cache string) ([]pythonCatalogEntry, error) {
	if !filepath.IsAbs(cache) {
		return nil, fmt.Errorf("uv cache must be absolute")
	}
	platform, err := runner.Platform()
	if err != nil {
		return nil, err
	}
	if err = u.VerifyVersion(ctx); err != nil {
		return nil, err
	}
	output := &boundedOutput{limit: 4 << 20}
	diagnostic := &boundedOutput{limit: 4096}
	code, err := executeBackend(ctx, runner.Process{Executable: u.Executable, Args: []string{"python", "list", "--only-downloads", "--all-versions", "--output-format", "json", "--no-config", "--offline", "--no-python-downloads", "--cache-dir", cache}, Environment: uvEnvironment(os.Environ()), Stdout: output, Stderr: diagnostic})
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("uv Python catalog failed (%d): %s", code, diagnostic.data)
	}
	var entries []pythonCatalogEntry
	if err = json.Unmarshal(output.data, &entries); err != nil {
		return nil, fmt.Errorf("invalid uv Python catalog: %w", err)
	}
	arch := "x86_64"
	if platform == "darwin-arm64" {
		arch = "aarch64"
	}

	selected := []pythonCatalogEntry{}
	for _, entry := range entries {
		if entry.Implementation != "cpython" || entry.Variant != "default" || entry.Arch != arch || entry.URL == "" {
			continue
		}
		if _, err := exactNodeVersion(entry.Version); err != nil && !config.IsPreview("python", entry.Version) {
			continue
		}
		selected = append(selected, entry)
	}
	return selected, nil
}

func (u UV) CatalogPython(ctx context.Context, cache string) ([]CatalogRelease, error) {
	entries, err := u.pythonEntries(ctx, cache)
	if err != nil {
		return nil, err
	}
	platform, err := runner.Platform()
	if err != nil {
		return nil, err
	}
	rows := []CatalogRelease{}
	for _, e := range entries {
		channel := "stable"
		if config.IsPreview("python", e.Version) {
			channel = "preview"
		}
		rows = append(rows, CatalogRelease{Tool: "python", Version: e.Version, Provider: "astral", Platform: platform, URL: e.URL, Channel: channel, Kind: "archive"})
	}
	return rows, nil
}
