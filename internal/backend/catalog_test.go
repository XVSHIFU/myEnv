package backend

import (
	"context"
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
	c := Catalog{javaMajor: 21, stableOnly: true, Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
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
