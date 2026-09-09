package backend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOfficialPythonPaginationAndSource(t *testing.T) {
	calls := 0
	c := Catalog{Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		body := `{"next":"index-windows-recent.json","versions":[]}`
		if calls == 2 {
			body = fmt.Sprintf(`{"versions":[{"id":"pythoncore-3.14-64","company":"PythonCore","sort-version":"3.14.7","url":"https://www.python.org/ftp/python/3.14.7/python-3.14.7-amd64.zip","hash":{"sha256":"%s"}},{"id":"pythonembed-3.14-64","company":"PythonCore","sort-version":"3.14.7","url":"https://www.python.org/ftp/python/3.14.7/python-3.14.7-embeddable-amd64.zip"}]}`, strings.Repeat("a", 64))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	rows, e := c.OfficialPython(context.Background(), "windows-amd64")
	if e != nil || len(rows) != 1 || calls != 2 {
		t.Fatalf("%+v %v calls=%d", rows, e, calls)
	}
	if ValidatePythonOrigin("https://www.python.org.evil/ftp/python/a.zip") == nil {
		t.Fatal("untrusted origin")
	}
	if _, e = c.OfficialPython(context.Background(), "linux-amd64-glibc"); e == nil {
		t.Fatal("pretended Linux official binary")
	}
}

func TestRustChannelPinsDateAndHash(t *testing.T) {
	c := Catalog{Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		body := fmt.Sprintf("date = '2026-09-08'\n[pkg.rust]\nversion = '1.100.0-nightly (abc 2026-09-07)'\n[pkg.rust.target.x86_64-pc-windows-msvc]\navailable = true\nurl = 'https://static.rust-lang.org/dist/2026-09-08/rust-nightly-x86_64-pc-windows-msvc.tar.gz'\nhash = '%s'\n", strings.Repeat("a", 64))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	lock, e := c.ResolveSDK(context.Background(), "rust", "nightly", "windows-amd64")
	if e != nil || lock.Version != "nightly-2026-09-08" || lock.RuntimeVersion != "1.100.0-nightly" {
		t.Fatalf("%+v %v", lock, e)
	}
	if _, e = c.RustChannel(context.Background(), "nightly-2026-09-09", "windows-amd64"); e == nil {
		t.Fatal("silently changed date")
	}
}
