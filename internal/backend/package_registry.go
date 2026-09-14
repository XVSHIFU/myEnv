package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	registryMetadataLimit = 32 << 20
	registrySearchLimit   = 2 << 20
	registryVersionLimit  = 1000
	registrySearchCount   = 20
)

// PackageRegistry reads only public npm/PyPI metadata. Client may carry the
// existing explicit CA/proxy transport; package-manager registry settings never
// change the official catalog's destination. No installer or shell is run.
type PackageRegistry struct{ Client *http.Client }

type RegistryPackageVersion struct {
	Ecosystem      string `json:"ecosystem"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	URL            string `json:"url"`
	State          string `json:"state"`
	Preview        bool   `json:"preview"`
	Yanked         bool   `json:"yanked"`
	Deprecated     string `json:"deprecated,omitempty"`
	RequiresPython string `json:"requires_python,omitempty"`
	RequiresNode   string `json:"requires_node,omitempty"`
	// Available means upstream has a distribution, not that this computer or
	// interpreter is compatible. The native manager still resolves compatibility.
	Available bool `json:"available"`
}

type RegistryPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Summary   string `json:"summary,omitempty"`
	Homepage  string `json:"homepage,omitempty"`
	Registry  string `json:"registry"`
	URL       string `json:"url"`
	State     string `json:"state"`
	// Latest is a stable, non-yanked version. Prefer upstream's latest tag;
	// previews never become an implicit update target.
	Latest            string                   `json:"latest,omitempty"`
	Versions          []RegistryPackageVersion `json:"versions"`
	VersionsTotal     int                      `json:"versions_total"`
	VersionsTruncated bool                     `json:"versions_truncated"`
	CheckedAt         string                   `json:"checked_at"`
}

type PackageSearchResult struct {
	Ecosystem string            `json:"ecosystem"`
	Query     string            `json:"query"`
	Mode      string            `json:"mode"`
	Packages  []RegistryPackage `json:"packages"`
	Total     int               `json:"total"`
	Truncated bool              `json:"truncated"`
}

var registryNodeName = regexp.MustCompile(`^(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`)
var registryPythonName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?$`)
var registryPythonSeparators = regexp.MustCompile(`[-_.]+`)

func registryEcosystem(ecosystem string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(ecosystem)) {
	case "node", "npm", "pnpm":
		return "node", nil
	case "python", "pip", "uv":
		return "python", nil
	default:
		return "", fmt.Errorf("unsupported package ecosystem %q", ecosystem)
	}
}

// NormalizeRegistryPackageName accepts names, never install specifications,
// local paths, aliases, URLs or command options.
func NormalizeRegistryPackageName(ecosystem, name string) (string, error) {
	eco, err := registryEcosystem(ecosystem)
	if err != nil {
		return "", err
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if len(name) == 0 || len(name) > 214 {
		return "", fmt.Errorf("invalid package name")
	}
	if eco == "node" {
		if !registryNodeName.MatchString(name) {
			return "", fmt.Errorf("invalid npm package name")
		}
		return name, nil
	}
	if !registryPythonName.MatchString(name) {
		return "", fmt.Errorf("invalid PyPI package name")
	}
	return registryPythonSeparators.ReplaceAllString(name, "-"), nil
}

func registryPackageBase(eco, name string) RegistryPackage {
	registry, page := "https://registry.npmjs.org", "https://www.npmjs.com/package/"+name
	if eco == "python" {
		registry, page = "https://pypi.org", "https://pypi.org/project/"+name+"/"
	}
	return RegistryPackage{Ecosystem: eco, Name: name, Registry: registry, URL: page, State: "found", Versions: []RegistryPackageVersion{}, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
}

func (c PackageRegistry) get(ctx context.Context, address string, limit int64) ([]byte, bool, error) {
	return c.getAccepted(ctx, address, limit, "application/json")
}

func (c PackageRegistry) getAccepted(ctx context.Context, address string, limit int64, accept string) ([]byte, bool, error) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || (parsed.Host != "registry.npmjs.org" && parsed.Host != "pypi.org") {
		return nil, false, fmt.Errorf("invalid official package registry URL")
	}
	base := c.Client
	if base == nil {
		base = NewNode().Client
	}
	client := *base
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.Host != parsed.Host || req.URL.User != nil {
			return fmt.Errorf("untrusted package registry redirect")
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "myenv-package-catalog/0.1")
	response, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("package registry request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, true, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("package registry HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, false, fmt.Errorf("package registry response failed: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, false, fmt.Errorf("package registry metadata exceeds %d bytes", limit)
	}
	return data, false, nil
}

// Search uses npm's public search endpoint. PyPI removed XML-RPC search; its
// supported API offers exact normalized-name lookup, which is reported as such.
func (c PackageRegistry) Search(ctx context.Context, ecosystem, query string) (PackageSearchResult, error) {
	eco, err := registryEcosystem(ecosystem)
	if err != nil {
		return PackageSearchResult{}, err
	}
	query = strings.TrimSpace(query)
	out := PackageSearchResult{Ecosystem: eco, Query: query, Mode: "search", Packages: []RegistryPackage{}}
	if query == "" || len(query) > 200 || strings.ContainsAny(query, "\x00\r\n") {
		return out, fmt.Errorf("package search requires 1–200 characters")
	}
	if eco == "python" {
		out.Mode = "exact"
		row, err := c.Package(ctx, eco, query)
		if err != nil {
			return out, err
		}
		if row.State == "found" {
			row.Versions = []RegistryPackageVersion{}
			out.Packages = append(out.Packages, row)
			out.Total = 1
		}
		return out, nil
	}
	data, _, err := c.get(ctx, "https://registry.npmjs.org/-/v1/search?size=20&text="+url.QueryEscape(query), registrySearchLimit)
	if err != nil {
		return out, err
	}
	var response struct {
		Objects []struct {
			Package struct {
				Name, Version, Description string
				Links                      struct{ Homepage string }
			}
		}
		Total int
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return out, fmt.Errorf("invalid npm search metadata: %w", err)
	}
	if response.Objects == nil || response.Total < 0 {
		return out, fmt.Errorf("incomplete npm search metadata")
	}
	out.Total = response.Total
	for _, item := range response.Objects {
		name, err := NormalizeRegistryPackageName(eco, item.Package.Name)
		if err != nil {
			continue
		}
		row := registryPackageBase(eco, name)
		row.Summary, row.Homepage = registryText(item.Package.Description, 600), registryHomepage(item.Package.Links.Homepage)
		if key := registryVersionKey(eco, item.Package.Version); key.valid && !key.preview {
			row.Latest = item.Package.Version
		}
		out.Packages = append(out.Packages, row)
		if len(out.Packages) == registrySearchCount {
			break
		}
	}
	out.Truncated = out.Total > len(out.Packages)
	return out, nil
}

func (c PackageRegistry) Package(ctx context.Context, ecosystem, name string) (RegistryPackage, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	eco, err := registryEcosystem(ecosystem)
	if err != nil {
		return RegistryPackage{}, err
	}
	name, err = NormalizeRegistryPackageName(eco, name)
	if err != nil {
		return RegistryPackage{}, err
	}
	if eco == "python" {
		return c.pythonPackage(ctx, name)
	}
	return c.nodePackage(ctx, name)
}

func (c PackageRegistry) Version(ctx context.Context, ecosystem, name, version string) (RegistryPackageVersion, error) {
	eco, err := registryEcosystem(ecosystem)
	if err != nil {
		return RegistryPackageVersion{}, err
	}
	name, err = NormalizeRegistryPackageName(eco, name)
	if err != nil {
		return RegistryPackageVersion{}, err
	}
	if !registryVersionKey(eco, version).valid {
		return RegistryPackageVersion{}, fmt.Errorf("invalid exact package version")
	}
	if eco == "python" {
		return c.pythonVersion(ctx, name, version)
	}
	return c.nodeVersion(ctx, name, version)
}

func registryFinishPackage(out *RegistryPackage, upstreamLatest string) {
	keys := make(map[string]registryVersionSortKey, len(out.Versions))
	for _, row := range out.Versions {
		keys[row.Version] = registryVersionKey(out.Ecosystem, row.Version)
	}
	sort.Slice(out.Versions, func(i, j int) bool {
		a, b := out.Versions[i], out.Versions[j]
		if a.Preview != b.Preview {
			return !a.Preview
		}
		if order := compareRegistryVersionKeys(out.Ecosystem, keys[a.Version], keys[b.Version]); order != 0 {
			return order > 0
		}
		return a.Version > b.Version
	})
	for _, row := range out.Versions {
		if !row.Available || row.Preview || row.Yanked {
			continue
		}
		if out.Latest == "" || row.Version == upstreamLatest {
			out.Latest = row.Version
		}
		if row.Version == upstreamLatest {
			break
		}
	}
	out.VersionsTotal = len(out.Versions)
	if len(out.Versions) > registryVersionLimit {
		// Keep the authoritative latest selectable even when an unusual package
		// publishes more than 1000 versions on other release channels.
		var latest *RegistryPackageVersion
		for i := range out.Versions[registryVersionLimit:] {
			row := out.Versions[registryVersionLimit+i]
			if row.Version == out.Latest {
				latest = &row
				break
			}
		}
		out.Versions = out.Versions[:registryVersionLimit]
		if latest != nil {
			out.Versions[registryVersionLimit-1] = *latest
		}
		out.VersionsTruncated = true
	}
}

func registryText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return value
}

func registryHomepage(value string) string {
	if len(value) > 2048 {
		return ""
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return ""
	}
	return value
}
