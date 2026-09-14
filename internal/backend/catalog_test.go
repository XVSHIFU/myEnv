package backend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type catalogTransport func(*http.Request) (*http.Response, error)

func (f catalogTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCatalogGoFullIndex(t *testing.T) {
	c := Catalog{Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("include") != "all" {
			t.Fatal("not requesting historical releases")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"version":"go1.9.0","stable":true,"files":[{"filename":"go1.9.0.windows-amd64.zip","os":"windows","arch":"amd64","kind":"archive"}]},{"version":"go1.10.0","stable":true,"files":[{"filename":"go1.10.0.windows-amd64.zip","os":"windows","arch":"amd64","kind":"archive"}]}]`))}, nil
	})}}
	rows, err := c.List(context.Background(), "go", "windows-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Version != "1.10.0" {
		t.Fatalf("incorrect catalog: %+v", rows)
	}
}

func TestCatalogPythonDoesNotClaimPlatformBinary(t *testing.T) {
	c := Catalog{Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"name":"Python 3.12.13","slug":"python-31213","is_published":true,"pre_release":false}]`))}, nil
	})}}
	rows, err := c.List(context.Background(), "python", "windows-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Kind != "release_page" || rows[0].Platform != "not_checked" {
		t.Fatal(rows)
	}
}
func TestSDKOriginAndRedirect(t *testing.T) {
	for _, u := range []string{"http://go.dev/dl/go.zip", "https://go.dev.evil/dl/go.zip", "https://evil/go.zip", "https://user:pass@go.dev/dl/go.zip"} {
		if ValidateSDKOrigin("go", u) == nil {
			t.Fatalf("accepted %s", u)
		}
	}
	if err := ValidateSDKOrigin("go", "https://go.dev/dl/go1.26.6.windows-amd64.zip"); err != nil {
		t.Fatal(err)
	}
	client := SDKClient(&http.Client{})
	req, _ := http.NewRequest("GET", "https://evil.invalid/payload", nil)
	if client.CheckRedirect(req, nil) == nil {
		t.Fatal("untrusted redirect accepted")
	}
}
func TestCatalogJavaFullVersion(t *testing.T) {
	if !releaseNewer("jdk-21.0.12.1+1", "jdk-21+35") {
		t.Fatal("Java build number outranked maintenance release")
	}
	c := Catalog{JavaMajor: 21, Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		text := `{"available_releases":[17,21]}`
		if strings.Contains(r.URL.Path, "assets/") {
			if !strings.Contains(r.URL.Path, "/21/ga") {
				t.Fatal(r.URL)
			}
			text = `[{"release_name":"jdk-21.0.12.1+1","binaries":[{"package":{"link":"https://github.com/adoptium/temurin21-binaries/releases/download/tag/archive.zip","checksum":"abc"}}]}]`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(text))}, nil
	})}}
	rows, err := c.List(context.Background(), "java", "windows-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Version != "jdk-21.0.12.1+1" {
		t.Fatal(rows)
	}
}

func TestCatalogJavaPreviewRequests(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(fmt.Sprint(preview), func(t *testing.T) {
			eaCalls := 0
			c := Catalog{JavaMajor: 21, IncludePreview: preview, Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
				body := `{"available_releases":[17,21]}`
				if strings.Contains(r.URL.Path, "assets/") {
					if !strings.Contains(r.URL.Path, "/21/") {
						t.Fatalf("queried unrequested Java major: %s", r.URL)
					}
					if strings.HasSuffix(r.URL.Path, "/ea") {
						eaCalls++
					}
					body = `[]`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			if _, err := c.List(context.Background(), "java", "linux-amd64-glibc"); err != nil {
				t.Fatal(err)
			}
			if (eaCalls != 0) != preview {
				t.Fatalf("preview=%v made %d EA requests", preview, eaCalls)
			}
		})
	}
}

func TestResolveJavaReleaseFamilies(t *testing.T) {
	for _, row := range []struct{ selector, major, kind, version string }{
		{"8", "8", "ga", "jdk8u422-b05"},
		{"1.8", "8", "ga", "jdk8u422-b05"},
		{"jdk1.8", "8", "ga", "jdk8u422-b05"},
		{"java8", "8", "ga", "jdk8u422-b05"},
		{"8u422", "8", "ga", "jdk8u422-b05"},
		{"21", "21", "ga", "jdk-21.0.2+13"},
		{"jdk-27+14-ea-beta", "27", "ea", "jdk-27+14-ea-beta"},
	} {
		t.Run(row.selector, func(t *testing.T) {
			c := Catalog{Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
				body := `{"available_releases":[8,17,21,27]}`
				if strings.Contains(r.URL.Path, "assets/") {
					if !strings.Contains(r.URL.Path, "/"+row.major+"/") {
						t.Fatalf("queried another major: %s", r.URL)
					}
					body = `[]`
					if strings.HasSuffix(r.URL.Path, "/"+row.kind) {
						body = fmt.Sprintf(`[{"release_name":%q,"binaries":[{"package":{"link":"https://github.com/adoptium/temurin%s-binaries/releases/download/tag/archive.tar.gz","checksum":%q}}]}]`, row.version, row.major, strings.Repeat("a", 64))
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			lock, err := c.ResolveSDK(context.Background(), "java", row.selector, "linux-amd64-glibc")
			if err != nil || lock.Version != row.version {
				t.Fatalf("resolved %+v: %v", lock, err)
			}
		})
	}
}
