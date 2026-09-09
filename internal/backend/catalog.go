package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// CatalogRelease describes upstream availability, not an installed environment.
type CatalogRelease struct {
	RuntimeVersion string `json:"runtime_version,omitempty"`
	Tool           string `json:"tool"`
	Version        string `json:"version"`
	Provider       string `json:"provider"`
	Platform       string `json:"platform"`
	URL            string `json:"url"`
	SHA256         string `json:"sha256,omitempty"`
	Channel        string `json:"channel"`
	Kind           string `json:"kind,omitempty"`
}

type Catalog struct {
	JavaMajor      int
	IncludePreview bool
	Client         *http.Client
	javaMajor      int
	stableOnly     bool
}

func (c Catalog) get(ctx context.Context, address string, limit int64) ([]byte, error) {
	client := c.Client
	if client == nil {
		client = NewNode().Client
	}
	secured := *client
	secured.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || req.URL.Scheme != "https" {
			return fmt.Errorf("invalid catalog redirect")
		}
		switch req.URL.Host {
		case "go.dev", "api.adoptium.net", "forge.rust-lang.org", "static.rust-lang.org", "nodejs.org", "www.python.org":
			return nil
		}
		return fmt.Errorf("untrusted catalog redirect")
	}
	client = &secured
	// Reuse bounded metadata reading and cancellation; no shell or installer runs.
	n := Node{Client: client, BaseURL: address}
	return n.metadata(ctx, "", limit)
}

func (c Catalog) List(ctx context.Context, tool, platform string) ([]CatalogRelease, error) {
	osName := ""
	switch platform {
	case "windows-amd64":
		osName = "windows"
	case "linux-amd64-glibc":
		osName = "linux"
	default:
		return nil, fmt.Errorf("unsupported catalog platform %q", platform)
	}
	var releases []CatalogRelease
	var err error
	switch tool {
	case "node":
		releases, err = c.nodeReleases(ctx, osName, platform)
		if err == nil && c.IncludePreview {
			for _, channel := range []string{"nightly", "rc", "v8-canary"} {
				rows, e := c.nodeChannelReleases(ctx, osName, platform, channel)
				if e != nil {
					return nil, e
				}
				releases = append(releases, rows...)
			}
		}
	case "python":
		releases, err = c.pythonReleases(ctx)
	case "go":
		releases, err = c.goReleases(ctx, osName, platform)
	case "java":
		releases, err = c.javaReleases(ctx, osName, platform)
	case "rust":
		releases, err = c.rustReleases(ctx, osName, platform)
		if err == nil && c.IncludePreview {
			for _, channel := range []string{"beta", "nightly"} {
				r, e := c.RustChannel(ctx, channel, platform)
				if e != nil {
					return nil, e
				}
				releases = append(releases, r)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported catalog tool %q", tool)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(releases, func(i, j int) bool { return releaseNewer(releases[i].Version, releases[j].Version) })
	return releases, nil
}

func (c Catalog) nodeReleases(ctx context.Context, osName, platform string) ([]CatalogRelease, error) {
	return c.nodeChannelReleases(ctx, osName, platform, "")
}
func (c Catalog) nodeChannelReleases(ctx context.Context, osName, platform, channel string) ([]CatalogRelease, error) {
	base := "https://nodejs.org/dist"
	if channel != "" {
		base = "https://nodejs.org/download/" + channel
	}
	data, err := c.get(ctx, base+"/index.json", 16<<20)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Version string
		Files   []string
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	key, target, extension := "linux-x64", "linux-x64", "tar.gz"
	if osName == "windows" {
		key, target, extension = "win-x64-zip", "win-x64", "zip"
	}
	out := []CatalogRelease{}
	for _, r := range rows {
		for _, file := range r.Files {
			if file == key {
				kind := "stable"
				if channel != "" {
					kind = "preview"
				}
				out = append(out, CatalogRelease{Tool: "node", Version: strings.TrimPrefix(r.Version, "v"), Provider: "Node.js", Platform: platform, URL: base + "/" + r.Version + "/node-" + r.Version + "-" + target + "." + extension, Channel: kind, Kind: "archive"})
				break
			}
		}
	}
	return out, nil
}

func (c Catalog) pythonReleases(ctx context.Context) ([]CatalogRelease, error) {
	data, err := c.get(ctx, "https://www.python.org/api/v2/downloads/release/?is_published=true", 8<<20)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Name, Slug string
		Published  bool `json:"is_published"`
		Preview    bool `json:"pre_release"`
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	out := []CatalogRelease{}
	for _, r := range rows {
		if !r.Published {
			continue
		}
		if strings.ContainsAny(r.Slug, "/\\") {
			return nil, fmt.Errorf("invalid Python release slug")
		}
		channel := "stable"
		if r.Preview {
			channel = "preview"
		}
		out = append(out, CatalogRelease{Tool: "python", Version: strings.TrimPrefix(r.Name, "Python "), Provider: "python.org CPython", Platform: "not_checked", URL: "https://www.python.org/downloads/release/" + r.Slug + "/", Channel: channel, Kind: "release_page"})
	}
	return out, nil
}

func (c Catalog) goReleases(ctx context.Context, osName, platform string) ([]CatalogRelease, error) {
	data, err := c.get(ctx, "https://go.dev/dl/?mode=json&include=all", 16<<20)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Version string
		Stable  bool
		Files   []struct{ Filename, OS, Arch, Kind, SHA256 string }
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	out := []CatalogRelease{}
	for _, r := range rows {
		for _, f := range r.Files {
			if f.OS != osName || f.Arch != "amd64" || f.Kind != "archive" {
				continue
			}
			if strings.ContainsAny(f.Filename, "/\\") {
				return nil, fmt.Errorf("invalid Go archive filename")
			}
			channel := "stable"
			if !r.Stable {
				channel = "preview"
			}
			out = append(out, CatalogRelease{Tool: "go", Version: strings.TrimPrefix(r.Version, "go"), Provider: "go.dev", Platform: platform, URL: "https://go.dev/dl/" + f.Filename, SHA256: f.SHA256, Channel: channel})
		}
	}
	return out, nil
}

func (c Catalog) javaReleases(ctx context.Context, osName, platform string) ([]CatalogRelease, error) {
	if c.javaMajor == 0 {
		c.javaMajor = c.JavaMajor
	}
	data, err := c.get(ctx, "https://api.adoptium.net/v3/info/available_releases", 1<<20)
	if err != nil {
		return nil, err
	}
	var available struct {
		Releases []int `json:"available_releases"`
	}
	if err = json.Unmarshal(data, &available); err != nil {
		return nil, err
	}
	out := []CatalogRelease{}
	for _, major := range available.Releases {
		if c.javaMajor != 0 && major != c.javaMajor {
			continue
		}
		for _, kind := range []string{"ga", "ea"} {
			if c.stableOnly && kind != "ga" {
				continue
			}
			for page := 0; ; page++ {
				if page >= 1000 {
					return nil, fmt.Errorf("Adoptium pagination exceeds bound; catalog incomplete")
				}
				address := fmt.Sprintf("https://api.adoptium.net/v3/assets/feature_releases/%d/%s?architecture=x64&image_type=jdk&jvm_impl=hotspot&heap_size=normal&os=%s&page=%d&page_size=20&vendor=eclipse", major, kind, osName, page)
				data, err = c.get(ctx, address, 8<<20)
				if err != nil {
					// Adoptium explicitly returns 404 for an empty release page.
					if strings.Contains(err.Error(), "HTTP 404") {
						break
					}
					return nil, err
				}
				var rows []struct {
					Name    string `json:"release_name"`
					Version struct {
						OpenJDK string `json:"openjdk_version"`
					} `json:"version_data"`
					Binaries []struct {
						Package struct{ Link, Checksum string }
					}
				}
				if err = json.Unmarshal(data, &rows); err != nil {
					return nil, err
				}
				for _, r := range rows {
					for _, b := range r.Binaries {
						u, e := url.Parse(b.Package.Link)
						if e != nil || u.Scheme != "https" || u.Host != "github.com" || !strings.HasPrefix(u.Path, "/adoptium/") {
							return nil, fmt.Errorf("unexpected Temurin release origin")
						}
						channel := "stable"
						if kind == "ea" {
							channel = "preview"
						}
						out = append(out, CatalogRelease{Tool: "java", Version: r.Name, RuntimeVersion: r.Version.OpenJDK, Provider: "Eclipse Temurin", Platform: platform, URL: b.Package.Link, SHA256: b.Package.Checksum, Channel: channel})
					}
				}
				if len(rows) < 20 {
					break
				}
			}
		}
	}
	return out, nil
}

func (c Catalog) rustReleases(ctx context.Context, osName, platform string) ([]CatalogRelease, error) {
	data, err := c.get(ctx, "https://forge.rust-lang.org/infra/archive-stable-version-installers.html", 8<<20)
	if err != nil {
		return nil, err
	}
	target := "x86_64-unknown-linux-gnu"
	if osName == "windows" {
		target = "x86_64-pc-windows-msvc"
	}
	pattern := regexp.MustCompile(`https://static\.rust-lang\.org/dist/(?:[0-9-]+/)?rust-([0-9]+\.[0-9]+\.[0-9]+)-` + regexp.QuoteMeta(target) + `\.tar\.gz`)
	out := []CatalogRelease{}
	seen := map[string]bool{}
	for _, m := range pattern.FindAllStringSubmatch(string(data), -1) {
		if seen[m[0]] {
			continue
		}
		seen[m[0]] = true
		out = append(out, CatalogRelease{Tool: "rust", Version: m[1], Provider: "Rust Project", Platform: platform, URL: m[0], Channel: "stable"})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("official Rust archive has no matching entries; page format may have changed")
	}
	return out, nil
}
