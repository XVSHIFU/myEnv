package backend

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"myenv/internal/config"
)

// Python.org publishes full Windows runtime ZIPs for its install manager.
// Embedded/free-threaded/test distributions are distinct and never substituted.
func (c Catalog) OfficialPython(ctx context.Context, platform string) ([]CatalogRelease, error) {
	if platform != "windows-amd64" {
		return nil, fmt.Errorf("python.org does not provide a managed Linux binary in this backend; select astral explicitly or use an existing distribution Python")
	}
	address := "https://www.python.org/ftp/python/index-windows.json"
	seen := map[string]bool{}
	out := []CatalogRelease{}
	for page := 0; address != ""; page++ {
		if page >= 32 || seen[address] {
			return nil, fmt.Errorf("invalid Python index pagination")
		}
		seen[address] = true
		if err := ValidatePythonOrigin(address); err != nil {
			return nil, err
		}
		data, err := c.get(ctx, address, 8<<20)
		if err != nil {
			return nil, err
		}
		var index struct {
			Next              string
			RequiresSignature bool `json:"requires_signature"`
			Versions          []struct {
				ID, Company, URL string
				Version          string `json:"sort-version"`
				Hash             map[string]string
			}
		}
		if err = json.Unmarshal(data, &index); err != nil {
			return nil, err
		}
		if index.RequiresSignature {
			return nil, fmt.Errorf("Python index requires a catalog signature; this backend cannot authenticate that signature and will not install")
		}
		for _, r := range index.Versions {
			if r.Company != "PythonCore" || !strings.HasPrefix(r.ID, "pythoncore-") || !strings.HasSuffix(r.ID, "-64") || strings.Contains(r.ID, "t-64") || !strings.HasSuffix(r.URL, "-amd64.zip") {
				continue
			}
			if err = ValidatePythonOrigin(r.URL); err != nil {
				return nil, err
			}
			hash, err := hex.DecodeString(r.Hash["sha256"])
			if err != nil || len(hash) != 32 {
				return nil, fmt.Errorf("missing official Python SHA256")
			}
			channel := "stable"
			if config.IsPreview("python", r.Version) {
				channel = "preview"
			}
			out = append(out, CatalogRelease{Tool: "python", Version: r.Version, Provider: "python.org", Platform: platform, URL: r.URL, SHA256: r.Hash["sha256"], Channel: channel, Kind: "archive"})
		}
		if index.Next == "" {
			break
		}
		base, _ := url.Parse(address)
		next, e := url.Parse(index.Next)
		if e != nil {
			return nil, e
		}
		address = base.ResolveReference(next).String()
	}
	return out, nil
}

func ValidatePythonOrigin(address string) error {
	u, e := url.Parse(address)
	if e != nil || u.Scheme != "https" || u.Host != "www.python.org" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/ftp/python/") || strings.Contains(u.Path, "..") || strings.Contains(u.Path, "\\") {
		return fmt.Errorf("untrusted official Python origin")
	}
	return nil
}

func (c Catalog) ResolveOfficialPython(ctx context.Context, selector, platform string) (config.RuntimeLock, error) {
	var result config.RuntimeLock
	constraint, e := config.ParseConstraint("python", selector)
	if e != nil {
		return result, e
	}
	rows, e := c.OfficialPython(ctx, platform)
	if e != nil {
		return result, e
	}
	for _, r := range rows {
		if !constraint.Contains(r.Version) || (r.Channel == "preview" && !config.IsPreview("python", selector)) {
			continue
		}
		if result.Version == "" || releaseNewer(r.Version, result.Version) {
			result = config.RuntimeLock{Version: r.Version, Backend: "python-official-v1", Evidence: "artifact", URL: r.URL, SHA256: r.SHA256}
		}
	}
	if result.Version == "" {
		return result, fmt.Errorf("no full official Python runtime matches %q on %s", selector, platform)
	}
	return result, nil
}

func PrepareOfficialPython(ctx context.Context, archive, directory, version string) (string, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	root := filepath.Join(directory, "python-runtime")
	if err := ExtractZIP(archive, root); err != nil {
		return "", err
	}
	entry := filepath.Join(root, "python.exe")
	// Some legacy official packages have a tools directory, like their NuGet layout.
	if _, err := os.Lstat(entry); os.IsNotExist(err) {
		entry = filepath.Join(root, "tools", "python.exe")
	}
	if err := verifyPython(ctx, entry, version); err != nil {
		return "", err
	}
	return entry, nil
}
