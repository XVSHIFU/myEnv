package cli

import (
	"context"
	"fmt"
	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type VersionQuery struct {
	Tool, Provider, Channel, Date, Search, Namespace, Certificate string
	Preview                                                       bool
	Major                                                         int
}
type VersionResult struct {
	Releases []backend.CatalogRelease `json:"releases"`
	Coverage string                   `json:"coverage"`
	Platform string                   `json:"platform"`
}

func QueryVersions(ctx context.Context, q VersionQuery) (VersionResult, error) {
	if !config.SupportedTool(q.Tool) {
		return VersionResult{}, fmt.Errorf("unsupported catalog tool %q", q.Tool)
	}
	if q.Provider != "" && (q.Tool != "python" || q.Provider != "python.org" && q.Provider != "astral") {
		return VersionResult{}, fmt.Errorf("invalid provider for catalog")
	}
	if q.Major < 0 || q.Major != 0 && q.Tool != "java" {
		return VersionResult{}, fmt.Errorf("--major applies only to Java")
	}
	if q.Channel != "" || q.Date != "" {
		if q.Tool != "rust" || q.Channel != "beta" && q.Channel != "nightly" {
			return VersionResult{}, fmt.Errorf("invalid Rust channel")
		}
		if q.Date != "" {
			if _, err := time.Parse("2006-01-02", q.Date); err != nil {
				return VersionResult{}, err
			}
		}
	}
	provider, preview, major, channel, date, search := q.Provider, q.Preview, q.Major, q.Channel, q.Date, q.Search
	if q.Tool == "python" && provider == "" {
		provider = "astral"
	}
	platform, err := runner.Platform()
	if err != nil {
		return VersionResult{}, err
	}
	certificate := q.Certificate
	if certificate == "" {
		certificate = os.Getenv("SSL_CERT_FILE")
	}
	client, err := backend.DownloadClient(certificate)
	if err != nil {
		return VersionResult{}, err
	}
	defer client.CloseIdleConnections()
	catalog := backend.Catalog{Client: client, IncludePreview: preview, JavaMajor: major, JavaRecent: true}
	var rows []backend.CatalogRelease
	if channel != "" || date != "" {
		selector := channel
		if date != "" {
			selector += "-" + date
		}
		r, e := catalog.RustChannel(ctx, selector, platform)
		err = e
		rows = []backend.CatalogRelease{r}
	} else if q.Tool == "python" && provider == "astral" {
		rows, err = astralVersions(ctx, q.Namespace, platform)
	} else if q.Tool == "python" && (provider == "python.org" || (provider == "" && platform == "windows-amd64")) {
		rows, err = catalog.OfficialPython(ctx, platform)
	} else {
		rows, err = catalog.List(ctx, q.Tool, platform)
	}
	if err != nil {
		return VersionResult{}, err
	}
	selected := []backend.CatalogRelease{}
	for _, r := range rows {
		if !preview && channel == "" && r.Channel != "stable" {
			continue
		}
		if !backend.CatalogReleaseMatches(r, search) {
			continue
		}
		selected = append(selected, r)
	}
	coverage := "current-platform upstream archives; preview installation requires an explicit preview version"
	if q.Tool == "python" {
		coverage = "python.org release pages only; not an installable current-platform catalog or the Astral/uv catalog"
		if platform == "windows-amd64" {
			coverage = "python.org full Windows x64 runtime archives; use --provider python.org to install; not the Astral/uv catalog"
		}
	}
	if q.Tool == "python" && provider == "astral" {
		coverage = "fixed uv installable CPython catalog for this platform"
	}
	if q.Tool == "rust" {
		coverage += "; beta/nightly current manifests with --preview; query historical dates with --channel and --date"
	}
	if q.Tool == "java" {
		coverage = "Eclipse Temurin / HotSpot JDK only, current platform; recent releases per major (up to 3 pages of 20 upstream rows per channel), not the full archive; exact historical IDs remain installable when available upstream"
	}
	return VersionResult{selected, coverage, platform}, nil
}
func astralVersions(ctx context.Context, namespace, platform string) ([]backend.CatalogRelease, error) {
	s, err := runtimeService(false, namespace)
	if err != nil {
		return nil, err
	}
	defer closeRuntimeService(s)
	directory := filepath.Join(s.Storage.Data, "backends")
	pointer, err := config.ReadInput(filepath.Join(directory, "uv-"+backend.UVVersion+"-"+platform))
	if err != nil {
		return nil, fmt.Errorf("Astral 目录需要已安装的 uv %s；请先通过 myenv 安装 Python，不会为查询自动安装后端: %w", backend.UVVersion, err)
	}
	entry, err := config.Within(directory, strings.TrimSpace(string(pointer)))
	if err != nil {
		return nil, err
	}
	return (backend.UV{Executable: entry}).CatalogPython(ctx, filepath.Join(s.Storage.Cache, "uv"))
}
