package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type registryNPMVersion struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	Deprecated  string `json:"deprecated"`
	Engines     struct {
		Node string `json:"node"`
	} `json:"engines"`
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

func registryNPMVersionRow(name, version string, metadata registryNPMVersion) RegistryPackageVersion {
	return RegistryPackageVersion{Ecosystem: "node", Name: name, Version: version,
		URL: "https://www.npmjs.com/package/" + name + "/v/" + url.PathEscape(version), State: "found",
		Preview:    registryVersionKey("node", version).preview,
		Deprecated: registryText(metadata.Deprecated, 600), RequiresNode: registryText(metadata.Engines.Node, 200),
		Available: metadata.Dist.Tarball != ""}
}

func (c PackageRegistry) nodePackage(ctx context.Context, name string) (RegistryPackage, error) {
	out := registryPackageBase("node", name)
	data, missing, err := c.getAccepted(ctx, "https://registry.npmjs.org/"+url.PathEscape(name), registryMetadataLimit, "application/vnd.npm.install-v1+json")
	if err != nil {
		return out, err
	}
	if missing {
		out.State = "not_found"
		return out, nil
	}
	var metadata struct {
		Name        string                     `json:"name"`
		Description string                     `json:"description"`
		Homepage    string                     `json:"homepage"`
		DistTags    map[string]string          `json:"dist-tags"`
		Versions    map[string]json.RawMessage `json:"versions"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return out, fmt.Errorf("invalid npm package metadata: %w", err)
	}
	canonical, err := NormalizeRegistryPackageName("node", metadata.Name)
	if err != nil || canonical != name || metadata.Versions == nil {
		return out, fmt.Errorf("npm returned mismatched or incomplete package metadata")
	}
	out.Summary, out.Homepage = registryText(metadata.Description, 600), registryHomepage(metadata.Homepage)
	for version, raw := range metadata.Versions {
		if !registryVersionKey("node", version).valid {
			continue
		}
		var release registryNPMVersion
		if err := json.Unmarshal(raw, &release); err != nil {
			return out, fmt.Errorf("invalid npm version metadata: %w", err)
		}
		if release.Version != "" && release.Version != version {
			return out, fmt.Errorf("npm returned mismatched version metadata")
		}
		out.Versions = append(out.Versions, registryNPMVersionRow(name, version, release))
	}
	registryFinishPackage(&out, metadata.DistTags["latest"])
	// The official abbreviated packument omits descriptions and homepages.
	// Request just the chosen stable release for its summary, not every historic
	// version's complete metadata/readme. No package archive is downloaded or run.
	if out.Latest != "" && out.Summary == "" {
		raw, missing, err := c.get(ctx, "https://registry.npmjs.org/"+url.PathEscape(name)+"/"+url.PathEscape(out.Latest), registrySearchLimit)
		if err != nil {
			return out, err
		}
		if missing {
			return out, fmt.Errorf("npm latest version disappeared during lookup; retry")
		}
		var latest registryNPMVersion
		if err := json.Unmarshal(raw, &latest); err != nil {
			return out, fmt.Errorf("invalid npm latest metadata: %w", err)
		}
		canonical, err := NormalizeRegistryPackageName("node", latest.Name)
		if err != nil || canonical != name || latest.Version != out.Latest {
			return out, fmt.Errorf("npm returned mismatched latest metadata")
		}
		out.Summary, out.Homepage = registryText(latest.Description, 600), registryHomepage(latest.Homepage)
	}
	return out, nil
}

func (c PackageRegistry) nodeVersion(ctx context.Context, name, version string) (RegistryPackageVersion, error) {
	out := registryNPMVersionRow(name, version, registryNPMVersion{})
	data, missing, err := c.get(ctx, "https://registry.npmjs.org/"+url.PathEscape(name)+"/"+url.PathEscape(version), registrySearchLimit)
	if err != nil {
		return out, err
	}
	if missing {
		out.State = "not_found"
		return out, nil
	}
	var release registryNPMVersion
	if err := json.Unmarshal(data, &release); err != nil {
		return out, fmt.Errorf("invalid npm version metadata: %w", err)
	}
	canonical, err := NormalizeRegistryPackageName("node", release.Name)
	if err != nil || canonical != name || release.Version != version {
		return out, fmt.Errorf("npm returned mismatched package/version metadata")
	}
	return registryNPMVersionRow(name, version, release), nil
}

type registryPythonFile struct {
	Yanked         bool   `json:"yanked"`
	URL            string `json:"url"`
	RequiresPython string `json:"requires_python"`
}

type registryPythonMetadata struct {
	Info struct {
		Name           string            `json:"name"`
		Version        string            `json:"version"`
		Summary        string            `json:"summary"`
		Homepage       string            `json:"home_page"`
		RequiresPython string            `json:"requires_python"`
		ProjectURLs    map[string]string `json:"project_urls"`
	} `json:"info"`
	Releases map[string][]registryPythonFile `json:"releases"`
	URLs     []registryPythonFile            `json:"urls"`
}

func registryPythonVersionRow(name, version string, files []registryPythonFile) RegistryPackageVersion {
	out := RegistryPackageVersion{Ecosystem: "python", Name: name, Version: version,
		URL: "https://pypi.org/project/" + name + "/" + url.PathEscape(version) + "/", State: "found",
		Preview: registryVersionKey("python", version).preview, Yanked: len(files) > 0}
	for _, file := range files {
		if file.URL == "" {
			continue
		}
		out.Available = true
		if !file.Yanked {
			out.Yanked = false
		}
	}
	if len(files) > 0 {
		out.RequiresPython = registryText(files[0].RequiresPython, 200)
		for _, file := range files[1:] {
			if file.RequiresPython != files[0].RequiresPython {
				out.RequiresPython = ""
				break
			}
		}
	}
	return out
}

func (c PackageRegistry) pythonPackage(ctx context.Context, name string) (RegistryPackage, error) {
	out := registryPackageBase("python", name)
	data, missing, err := c.get(ctx, "https://pypi.org/pypi/"+url.PathEscape(name)+"/json", registryMetadataLimit)
	if err != nil {
		return out, err
	}
	if missing {
		out.State = "not_found"
		return out, nil
	}
	var metadata registryPythonMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return out, fmt.Errorf("invalid PyPI package metadata: %w", err)
	}
	canonical, err := NormalizeRegistryPackageName("python", metadata.Info.Name)
	if err != nil || canonical != name || metadata.Releases == nil {
		return out, fmt.Errorf("PyPI returned mismatched or incomplete package metadata")
	}
	out.Summary = registryText(metadata.Info.Summary, 600)
	out.Homepage = registryHomepage(metadata.Info.Homepage)
	if out.Homepage == "" {
		out.Homepage = registryHomepage(metadata.Info.ProjectURLs["Homepage"])
	}
	for version, files := range metadata.Releases {
		if !registryVersionKey("python", version).valid {
			continue
		}
		out.Versions = append(out.Versions, registryPythonVersionRow(name, version, files))
	}
	registryFinishPackage(&out, metadata.Info.Version)
	return out, nil
}

func (c PackageRegistry) pythonVersion(ctx context.Context, name, version string) (RegistryPackageVersion, error) {
	out := registryPythonVersionRow(name, version, nil)
	data, missing, err := c.get(ctx, "https://pypi.org/pypi/"+url.PathEscape(name)+"/"+url.PathEscape(version)+"/json", registrySearchLimit)
	if err != nil {
		return out, err
	}
	if missing {
		out.State = "not_found"
		return out, nil
	}
	var metadata registryPythonMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return out, fmt.Errorf("invalid PyPI version metadata: %w", err)
	}
	canonical, err := NormalizeRegistryPackageName("python", metadata.Info.Name)
	// PyPI may normalize 1.0-1 to 1.0.post1, so compare its parsed version.
	if err != nil || canonical != name || !registryVersionKey("python", metadata.Info.Version).valid || compareRegistryVersions("python", metadata.Info.Version, version) != 0 {
		return out, fmt.Errorf("PyPI returned mismatched package/version metadata")
	}
	out = registryPythonVersionRow(name, metadata.Info.Version, metadata.URLs)
	out.RequiresPython = registryText(metadata.Info.RequiresPython, 200)
	return out, nil
}
