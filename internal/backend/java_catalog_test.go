package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func javaFixture(name string, major, security, build int, pre string) map[string]any {
	return map[string]any{
		"release_name": name, "release_type": "ga",
		"version_data": map[string]any{"major": major, "minor": 0, "security": security, "build": build, "pre": pre, "openjdk_version": fmt.Sprintf("%d.0.%d+%d", major, security, build)},
		"binaries":     []any{map[string]any{"package": map[string]string{"link": "https://github.com/adoptium/temurin" + strconv.Itoa(major) + "-binaries/releases/download/" + name + "/archive.zip", "checksum": strings.Repeat("a", 64)}}},
	}
}

func TestJavaCatalogStableLabelsAndOrdering(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(fmt.Sprint(preview), func(t *testing.T) {
			c := Catalog{JavaRecent: true, IncludePreview: preview, Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
				var data any = map[string]any{"available_releases": []int{8, 21, 27}, "available_lts_releases": []int{8, 21}}
				if strings.Contains(r.URL.Path, "assets/") {
					if q := r.URL.Query(); q.Get("sort_order") != "DESC" || q.Get("sort_method") != "DEFAULT" || q.Get("page_size") != "20" {
						t.Errorf("invalid upstream version-order request: %s", r.URL)
					}
					data = []any{}
					if strings.Contains(r.URL.Path, "/8/ga") {
						data = []any{javaFixture("jdk-2026-09-14-13-26-beta", 8, 502, 99, ""), javaFixture("jdk8u502-b09", 8, 502, 9, ""), javaFixture("jdk8u502-b10", 8, 502, 10, "")}
					} else if strings.Contains(r.URL.Path, "/21/ga") {
						data = []any{javaFixture("jdk-21.0.8+9", 21, 8, 9, ""), javaFixture("jdk-21.0.9+1", 21, 9, 1, "")}
					} else if strings.Contains(r.URL.Path, "/27/ga") {
						// Even a GA endpoint or harmless-looking ID does not override pre metadata.
						data = []any{javaFixture("jdk-27+1", 27, 0, 1, "ea")}
					}
				}
				body, _ := json.Marshal(data)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
			})}}
			rows, err := c.List(context.Background(), "java", "windows-amd64")
			if err != nil {
				t.Fatal(err)
			}
			want := 4
			if preview {
				want = 6
			}
			if len(rows) != want || rows[0].Version != "jdk-21.0.9+1" || rows[2].Version != "jdk8u502-b10" {
				t.Fatalf("wrong stable order/count: %+v", rows)
			}
			if rows[2].DisplayVersion != "Java 8 (1.8) · 8u502" || rows[2].Major != 8 || !rows[2].LTS {
				t.Fatalf("missing familiar Java metadata: %+v", rows[2])
			}
			if preview && (rows[4].Channel != "preview" || rows[5].Channel != "preview") {
				t.Fatal("preview rows not separated")
			}
			encoded, _ := json.Marshal(rows[2])
			if !strings.Contains(string(encoded), `"version":"jdk8u502-b10"`) || strings.Contains(string(encoded), "javaSortVersion") {
				t.Fatalf("installation identity or JSON polluted: %s", encoded)
			}
		})
	}
}

func TestJavaCatalogRecentBoundKeepsHistoricalResolution(t *testing.T) {
	for _, recent := range []bool{true, false} {
		calls := 0
		c := Catalog{JavaMajor: 8, JavaRecent: recent, Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
			var data any = map[string]any{"available_releases": []int{8}}
			if strings.Contains(r.URL.Path, "assets/") {
				calls++
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				rows := []any{}
				if page < 3 {
					for i := 0; i < javaPageSize; i++ {
						rows = append(rows, javaFixture(fmt.Sprintf("jdk8u502-b%d-beta", page*javaPageSize+i), 8, 502, i, "beta"))
					}
				} else {
					rows = append(rows, javaFixture("jdk8u422-b05", 8, 422, 5, ""))
				}
				data = rows
			}
			body, _ := json.Marshal(data)
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
		})}}
		if recent {
			rows, err := c.List(context.Background(), "java", "windows-amd64")
			if err != nil || calls != javaRecentPages || len(rows) != 0 {
				t.Fatalf("recent limit: %d calls, %+v, %v", calls, rows, err)
			}
		} else {
			lock, err := c.ResolveSDK(context.Background(), "java", "jdk8u422-b05", "windows-amd64")
			if err != nil || lock.Version != "jdk8u422-b05" || calls != 4 {
				t.Fatalf("historical resolution was limited: %+v %v calls=%d", lock, err, calls)
			}
		}
	}
}

func TestJavaCatalogAliasSearch(t *testing.T) {
	java8 := CatalogRelease{Tool: "java", Version: "jdk8u502-b07", RuntimeVersion: "8.0.502+7", DisplayVersion: "Java 8 (1.8) · 8u502", Major: 8, Provider: "Eclipse Temurin"}
	java21 := CatalogRelease{Tool: "java", Version: "jdk-21.0.8+9", RuntimeVersion: "21.0.8+9", DisplayVersion: "Java 21 · 21.0.8", Major: 21, Provider: "Eclipse Temurin"}
	for _, search := range []string{"1.8", "jdk1.8", "java8", "java 8", "JDK 1.8", "8u", "8", "8u50", "temurin 1.8"} {
		if !CatalogReleaseMatches(java8, search) || CatalogReleaseMatches(java21, search) {
			t.Errorf("alias did not select only Java 8: %q", search)
		}
	}
	for _, search := range []string{"jdk", "java", "temurin"} {
		if !CatalogReleaseMatches(java8, search) || !CatalogReleaseMatches(java21, search) {
			t.Errorf("common search rejected %q", search)
		}
	}
	if !CatalogReleaseMatches(java21, "21.0.8") || CatalogReleaseMatches(java8, "21.0.8") {
		t.Fatal("runtime version search failed")
	}
	if !CatalogReleaseMatches(CatalogRelease{Tool: "go", Version: "1.8.5"}, "1.8") {
		t.Fatal("Java alias changed another ecosystem")
	}
}

func TestJavaCatalogUnsupportedPreviewIsReadOnly(t *testing.T) {
	const dated = "jdk-2026-09-14-13-26-beta"
	const supported = "jdk8u-2026-09-14-13-26-beta"
	c := Catalog{JavaMajor: 8, IncludePreview: true, JavaRecent: true, Client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		var data any = map[string]any{"available_releases": []int{8}}
		if strings.Contains(r.URL.Path, "assets/") {
			data = []any{}
			if strings.HasSuffix(r.URL.Path, "/ga") {
				data = []any{javaFixture(dated, 8, 502, 7, "beta"), javaFixture(supported, 8, 502, 7, "beta"), javaFixture("jdk8u502-b07", 8, 502, 7, "")}
			}
		}
		body, _ := json.Marshal(data)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	rows, err := c.List(context.Background(), "java", "windows-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("lost discovery records: %+v", rows)
	}
	for _, r := range rows {
		if r.Version == dated {
			if r.Kind != "release_page" || r.Channel != "preview" || !strings.HasSuffix(r.URL, "/releases/tag/"+dated) || r.SHA256 != "" {
				t.Fatalf("unsupported dated ID still offered as installable: %+v", r)
			}
		} else if r.Kind != "archive" || r.SHA256 == "" || !strings.Contains(r.URL, "/releases/download/") {
			t.Fatalf("supported exact ID changed: %+v", r)
		}
	}
	lock, err := c.ResolveSDK(context.Background(), "java", "8", "windows-amd64")
	if err != nil || lock.Version != "jdk8u502-b07" {
		t.Fatalf("stable family selected a read-only preview: %+v %v", lock, err)
	}
}
