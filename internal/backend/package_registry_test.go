package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func registryTestClient(t *testing.T, handler func(*http.Request) (int, string)) *http.Client {
	t.Helper()
	return &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Scheme != "https" || (r.URL.Host != "registry.npmjs.org" && r.URL.Host != "pypi.org") {
			t.Fatalf("untrusted destination: %s", r.URL)
		}
		if r.Header.Get("User-Agent") == "" || r.Header.Get("Authorization") != "" {
			t.Fatalf("unexpected request headers: %v", r.Header)
		}
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 30*time.Second {
			t.Fatal("catalog request has no bounded deadline")
		}
		status, body := handler(r)
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})}
}

func TestPackageRegistryNPMOfficialIdentityLatestAndVersions(t *testing.T) {
	c := PackageRegistry{Client: registryTestClient(t, func(r *http.Request) (int, string) {
		if r.URL.EscapedPath() != "/@example%2Ftool" {
			t.Fatalf("scope not escaped: %s", r.URL.EscapedPath())
		}
		return 200, `{"name":"@example/tool","description":"Tool\nsummary","homepage":"javascript:alert(1)","dist-tags":{"latest":"2.9.0"},"versions":{"2.9.0":{"version":"2.9.0","engines":{"node":">=22"},"dist":{"tarball":"https://registry.npmjs.org/tool.tgz"}},"2.10.0":{"version":"2.10.0","dist":{"tarball":"https://registry.npmjs.org/tool.tgz"}},"3.0.0-beta.2":{"version":"3.0.0-beta.2","dist":{"tarball":"https://registry.npmjs.org/tool.tgz"}},"1.0.0":{"deprecated":"Use a newer version","dist":{"tarball":"https://registry.npmjs.org/tool.tgz"}}}}`
	})}
	out, err := c.Package(context.Background(), "pnpm", "@example/tool")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != "found" || out.Latest != "2.9.0" || out.Homepage != "" || out.Summary != "Tool summary" || out.Ecosystem != "node" || out.Registry != "https://registry.npmjs.org" {
		t.Fatalf("bad metadata: %+v", out)
	}
	if len(out.Versions) != 4 || out.Versions[3].Version != "3.0.0-beta.2" || !out.Versions[3].Preview || out.Versions[0].Version != "2.10.0" || out.Versions[1].RequiresNode != ">=22" || out.Versions[2].Deprecated == "" {
		t.Fatalf("bad versions: %+v", out.Versions)
	}
}

func TestPackageRegistryPythonLatestYankedAndExactSearch(t *testing.T) {
	c := PackageRegistry{Client: registryTestClient(t, func(r *http.Request) (int, string) {
		if r.URL.Path != "/pypi/example-tool/json" {
			t.Fatalf("unexpected PyPI search: %s", r.URL)
		}
		return 200, `{"info":{"name":"Example_Tool","version":"3.0rc1","summary":"Example tool","project_urls":{"Homepage":"https://example.org"}},"releases":{"3.0rc1":[{"url":"https://files.pythonhosted.org/3.whl"}],"2.1":[{"url":"https://files.pythonhosted.org/2.whl","yanked":true}],"2.0.post1":[{"url":"https://files.pythonhosted.org/a.whl","yanked":true},{"url":"https://files.pythonhosted.org/b.whl","requires_python":">=3.9"}],"2.0":[],"1.9":[{"url":"https://files.pythonhosted.org/1.whl"}]}}`
	})}
	out, err := c.Package(context.Background(), "pip", "Example.Tool")
	if err != nil {
		t.Fatal(err)
	}
	if out.Latest != "2.0.post1" || out.Name != "example-tool" || out.Homepage != "https://example.org" {
		t.Fatalf("bad metadata: %+v", out)
	}
	if !out.Versions[4].Preview || !out.Versions[0].Yanked || out.Versions[1].Yanked || out.Versions[2].Available {
		t.Fatalf("bad availability: %+v", out.Versions)
	}
	search, err := c.Search(context.Background(), "uv", "Example_Tool")
	if err != nil || search.Mode != "exact" || search.Total != 1 || len(search.Packages) != 1 || len(search.Packages[0].Versions) != 0 {
		t.Fatalf("misrepresented search: %+v, %v", search, err)
	}
}

func TestPackageRegistryNPMAbbreviatedCatalogAndSummary(t *testing.T) {
	requests := 0
	c := PackageRegistry{Client: registryTestClient(t, func(r *http.Request) (int, string) {
		requests++
		if r.URL.Path == "/example" {
			if r.Header.Get("Accept") != "application/vnd.npm.install-v1+json" {
				t.Fatal("requested historic full metadata")
			}
			return 200, `{"name":"example","dist-tags":{"latest":"1.0.0"},"versions":{"1.0.0":{"version":"1.0.0","dist":{"tarball":"https://registry.npmjs.org/example.tgz"}}}}`
		}
		if r.URL.Path != "/example/1.0.0" {
			t.Fatal(r.URL)
		}
		return 200, `{"name":"example","version":"1.0.0","description":"Current summary","homepage":"https://example.org"}`
	})}
	out, err := c.Package(context.Background(), "node", "example")
	if err != nil || requests != 2 || out.Summary != "Current summary" || out.Homepage != "https://example.org" {
		t.Fatalf("bad abbreviated metadata: %+v %v requests=%d", out, err, requests)
	}
}

func TestPackageRegistryNPMSearchBoundAndEscaping(t *testing.T) {
	c := PackageRegistry{Client: registryTestClient(t, func(r *http.Request) (int, string) {
		if r.URL.Path != "/-/v1/search" || r.URL.Query().Get("text") != "format & lint" || r.URL.Query().Get("size") != "20" {
			t.Fatalf("wrong query: %s", r.URL)
		}
		rows := make([]map[string]any, 30)
		for i := range rows {
			rows[i] = map[string]any{"package": map[string]string{"name": fmt.Sprintf("tool-%d", i), "version": "1.2.3", "description": "A tool"}}
		}
		data, _ := json.Marshal(map[string]any{"total": 300, "objects": rows})
		return 200, string(data)
	})}
	out, err := c.Search(context.Background(), "npm", "format & lint")
	if err != nil || out.Mode != "search" || out.Total != 300 || len(out.Packages) != 20 || !out.Truncated {
		t.Fatalf("bad bounded search: %+v, %v", out, err)
	}
}

func TestPackageRegistryNotFoundIsNotNetworkFailure(t *testing.T) {
	for _, eco := range []string{"node", "python"} {
		for _, status := range []int{404, 403, 429, 500} {
			t.Run(fmt.Sprintf("%s-%d", eco, status), func(t *testing.T) {
				c := PackageRegistry{Client: registryTestClient(t, func(*http.Request) (int, string) { return status, `{"error":"missing"}` })}
				out, err := c.Package(context.Background(), eco, "example")
				if status == 404 {
					if err != nil || out.State != "not_found" {
						t.Fatalf("404: %+v, %v", out, err)
					}
				} else if err == nil {
					t.Fatalf("HTTP %d was reported as a package result", status)
				}
				v, err := c.Version(context.Background(), eco, "example", "1.0.0")
				if status == 404 {
					if err != nil || v.State != "not_found" || v.Available {
						t.Fatalf("404 version: %+v, %v", v, err)
					}
				} else if err == nil {
					t.Fatalf("HTTP %d version was accepted", status)
				}
			})
		}
	}
}

func TestPackageRegistryRejectsInvalidIdentityBeforeRequest(t *testing.T) {
	c := PackageRegistry{Client: registryTestClient(t, func(*http.Request) (int, string) { t.Fatal("invalid input made a request"); return 200, "" })}
	for _, name := range []string{"../example", "-r", "pkg@latest", "https://evil.invalid/tool", "pkg\x00x", "@scope/../../evil", "pkg;whoami", "pkg[extra]"} {
		for _, eco := range []string{"node", "python"} {
			if _, err := c.Package(context.Background(), eco, name); err == nil {
				t.Fatalf("accepted %s name %q", eco, name)
			}
		}
	}
	for _, version := range []string{"latest", "1.0.0 --global", "../2.0.0", "file:evil", "1.0.0\nfoo"} {
		if _, err := c.Version(context.Background(), "node", "example", version); err == nil {
			t.Fatalf("accepted version %q", version)
		}
	}
}

func TestPackageRegistryExactVersionsCheckCanonicalIdentity(t *testing.T) {
	c := PackageRegistry{Client: registryTestClient(t, func(r *http.Request) (int, string) {
		if r.URL.Host == "registry.npmjs.org" {
			return 200, `{"name":"example","version":"1.0.0","engines":{"node":">=20"},"dist":{"tarball":"https://registry.npmjs.org/example.tgz"}}`
		}
		return 200, `{"info":{"name":"Example","version":"1.0.post1","requires_python":">=3.10"},"urls":[{"url":"https://files.pythonhosted.org/example.whl"}]}`
	})}
	node, err := c.Version(context.Background(), "npm", "example", "1.0.0")
	if err != nil || !node.Available || node.RequiresNode != ">=20" {
		t.Fatalf("bad exact npm: %+v %v", node, err)
	}
	python, err := c.Version(context.Background(), "python", "example", "1.0-1")
	if err != nil || !python.Available || python.Version != "1.0.post1" || python.RequiresPython != ">=3.10" {
		t.Fatalf("bad exact Python: %+v %v", python, err)
	}
	if _, err := c.Version(context.Background(), "node", "other-package", "1.0.0"); err == nil {
		t.Fatal("accepted mismatched npm identity")
	}
	if _, err := c.Version(context.Background(), "python", "example", "1.0"); err == nil {
		t.Fatal("accepted mismatched PyPI version")
	}
}

func TestPackageRegistryRejectsMalformedMetadata(t *testing.T) {
	for _, body := range []string{"<html>Service unavailable</html>", "{}", `{"name":"another","versions":{}}`, `{"info":{"name":"another"},"releases":{}}`} {
		c := PackageRegistry{Client: registryTestClient(t, func(*http.Request) (int, string) { return 200, body })}
		for _, eco := range []string{"node", "python"} {
			if _, err := c.Package(context.Background(), eco, "example"); err == nil {
				t.Fatalf("accepted malformed %s metadata: %s", eco, body)
			}
		}
	}
}

func TestPackageRegistryCancellationAndResponseBound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := PackageRegistry{Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })}}
	if _, err := c.Package(ctx, "node", "example"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	c.Client = registryTestClient(t, func(*http.Request) (int, string) { return 200, strings.Repeat("x", 65) })
	if _, _, err := c.get(context.Background(), "https://registry.npmjs.org/example", 64); err == nil {
		t.Fatal("oversized metadata accepted")
	}
}

func TestPackageRegistryRedirectStaysOnOfficialOrigin(t *testing.T) {
	for _, destination := range []string{"https://evil.invalid/example", "http://registry.npmjs.org/example", "https://pypi.org/example", "https://user:pass@registry.npmjs.org/example"} {
		requests := 0
		c := PackageRegistry{Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{destination}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		})}}
		if _, err := c.Package(context.Background(), "node", "example"); err == nil || requests != 1 {
			t.Fatalf("followed forbidden redirect %s: %v (%d requests)", destination, err, requests)
		}
	}
}

func TestPackageRegistryVersionOrder(t *testing.T) {
	cases := []struct {
		eco      string
		versions []string
	}{
		{"node", []string{"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.2.0", "1.10.0", "2.0.0"}},
		{"python", []string{"1.0.dev1", "1.0a1.dev1", "1.0a1", "1.0b1", "1.0rc1", "1.0", "1.0+abc.1", "1.0+abc.2", "1.0+abc.10", "1.0+1", "1.0.post1.dev1", "1.0.post1", "1.0.post2", "1.2", "1.10", "1!0.1"}},
	}
	for _, tc := range cases {
		for i := 1; i < len(tc.versions); i++ {
			a, b := tc.versions[i-1], tc.versions[i]
			if !registryVersionKey(tc.eco, a).valid || compareRegistryVersions(tc.eco, a, b) >= 0 {
				t.Fatalf("%s: expected %s < %s", tc.eco, a, b)
			}
		}
	}
	for _, pair := range [][2]string{{"1.0-1", "1.0.post1"}, {"1.0pre1", "1.0rc1"}, {"1.0.0", "1.0"}, {"v1.0", "1.0"}, {"1.0rev2", "1.0.post2"}} {
		if compareRegistryVersions("python", pair[0], pair[1]) != 0 {
			t.Fatalf("normalization differs: %v", pair)
		}
	}
	if compareRegistryVersions("node", "1.0.0+build1", "1.0.0+build2") != 0 {
		t.Fatal("SemVer build metadata changed precedence")
	}
	if registryVersionKey("python", "1.0.post1").preview || !registryVersionKey("python", "1.0.post1.dev1").preview {
		t.Fatal("post/dev channel classification incorrect")
	}
}

func TestPackageRegistryVersionOutputIsBoundedAndKeepsLatest(t *testing.T) {
	out := registryPackageBase("node", "example")
	for i := 0; i < 1100; i++ {
		out.Versions = append(out.Versions, RegistryPackageVersion{Version: fmt.Sprintf("1.%d.0", i), Available: true})
	}
	registryFinishPackage(&out, "1.0.0")
	if len(out.Versions) != 1000 || !out.VersionsTruncated || out.VersionsTotal != 1100 || out.Latest != "1.0.0" || out.Versions[999].Version != "1.0.0" {
		t.Fatalf("output bound/latest lost: %+v", out)
	}
}
