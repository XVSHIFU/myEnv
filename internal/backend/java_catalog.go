package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"myenv/internal/config"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const javaPageSize = 20
const javaRecentPages = 3

type javaVersionData struct {
	OpenJDK  string `json:"openjdk_version"`
	Major    int    `json:"major"`
	Minor    int    `json:"minor"`
	Security int    `json:"security"`
	Patch    int    `json:"patch"`
	Build    int    `json:"build"`
	Pre      string `json:"pre"`
}

func (c Catalog) javaReleases(ctx context.Context, osName, platform string) ([]CatalogRelease, error) {
	data, err := c.get(ctx, "https://api.adoptium.net/v3/info/available_releases", 1<<20)
	if err != nil {
		return nil, err
	}
	var available struct {
		Releases []int `json:"available_releases"`
		LTS      []int `json:"available_lts_releases"`
	}
	if err = json.Unmarshal(data, &available); err != nil {
		return nil, err
	}
	if len(available.Releases) > 64 {
		return nil, fmt.Errorf("unexpected Adoptium major count")
	}
	majors := []int{}
	seen := map[int]bool{}
	lts := map[int]bool{}
	for _, major := range available.LTS {
		lts[major] = true
	}
	for _, major := range available.Releases {
		if major < 1 || major > 999 || seen[major] || c.JavaMajor != 0 && major != c.JavaMajor {
			continue
		}
		seen[major] = true
		majors = append(majors, major)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(majors)))
	// At most four metadata requests run concurrently. No worker mutates shared
	// release data, and cancellation joins all workers before returning.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([][]CatalogRelease, len(majors))
	errs := make([]error, len(majors))
	jobs := make(chan int)
	var workers sync.WaitGroup
	for worker := 0; worker < min(4, len(majors)); worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := range jobs {
				results[i], errs[i] = c.javaMajorReleases(ctx, osName, platform, majors[i], lts[majors[i]])
				if errs[i] != nil {
					cancel()
				}
			}
		}()
	}
	for i := range majors {
		jobs <- i
	}
	close(jobs)
	workers.Wait()
	// Prefer the original failure to sibling requests canceled by it.
	for _, e := range errs {
		if e != nil && !errors.Is(e, context.Canceled) {
			return nil, e
		}
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	out := []CatalogRelease{}
	for _, rows := range results {
		out = append(out, rows...)
	}
	return out, nil
}

func (c Catalog) javaMajorReleases(ctx context.Context, osName, platform string, major int, lts bool) ([]CatalogRelease, error) {
	out := []CatalogRelease{}
	seen := map[string]bool{}
	for _, kind := range []string{"ga", "ea"} {
		if kind == "ea" && !c.IncludePreview {
			continue
		}
		kept := 0
		for page := 0; ; page++ {
			if page >= 1000 {
				return nil, fmt.Errorf("Adoptium pagination exceeds bound; catalog incomplete")
			}
			address := fmt.Sprintf("https://api.adoptium.net/v3/assets/feature_releases/%d/%s?architecture=x64&image_type=jdk&jvm_impl=hotspot&heap_size=normal&os=%s&page=%d&page_size=%d&vendor=eclipse&sort_method=DEFAULT&sort_order=DESC", major, kind, osName, page, javaPageSize)
			data, err := c.get(ctx, address, 8<<20)
			if err != nil {
				if strings.Contains(err.Error(), "HTTP 404") {
					break
				}
				return nil, err
			}
			var rows []struct {
				Name     string          `json:"release_name"`
				Type     string          `json:"release_type"`
				Version  javaVersionData `json:"version_data"`
				Binaries []struct {
					Package struct{ Link, Checksum string }
				}
			}
			if err = json.Unmarshal(data, &rows); err != nil {
				return nil, err
			}
			for _, r := range rows {
				if r.Version.Major != 0 && r.Version.Major != major {
					return nil, fmt.Errorf("Temurin release has an unexpected Java major")
				}
				channel := "stable"
				if kind == "ea" || r.Type == "ea" || r.Version.Pre != "" || javaPreviewName.MatchString(r.Name) || javaPreviewName.MatchString(r.Version.OpenJDK) {
					channel = "preview"
				}
				if channel == "preview" && !c.IncludePreview {
					continue
				}
				releaseKind := "archive"
				if _, err := config.ParseConstraint("java", r.Name); err != nil || channel == "preview" && !config.IsPreview("java", r.Name) {
					// The upstream catalog can include dated or otherwise unsupported
					// identifiers. Preserve discovery without offering a broken install.
					releaseKind = "release_page"
				}
				for _, b := range r.Binaries {
					u, e := url.Parse(b.Package.Link)
					if e != nil || u.Scheme != "https" || u.User != nil || u.Host != "github.com" || !strings.HasPrefix(u.Path, "/adoptium/") {
						return nil, fmt.Errorf("unexpected Temurin release origin")
					}
					key := r.Name + "\x00" + b.Package.Link
					if seen[key] {
						continue
					}
					seen[key] = true
					sortVersion := ""
					if r.Version.Major > 0 {
						sortVersion = fmt.Sprintf("%d.%d.%d.%d+%d", r.Version.Major, r.Version.Minor, r.Version.Security, r.Version.Patch, r.Version.Build)
					}
					address, checksum := b.Package.Link, b.Package.Checksum
					if releaseKind == "release_page" {
						// Link to the provider's release record, not an archive that this
						// configuration grammar cannot select for installation.
						path, _, ok := strings.Cut(u.Path, "/releases/download/")
						if !ok {
							return nil, fmt.Errorf("unexpected Temurin release record path")
						}
						u.Path, u.RawPath, u.RawQuery, u.Fragment = path+"/releases/tag/"+r.Name, "", "", ""
						address, checksum = u.String(), ""
					}
					out = append(out, CatalogRelease{javaSortVersion: sortVersion, Tool: "java", Version: r.Name, RuntimeVersion: r.Version.OpenJDK, DisplayVersion: javaDisplayVersion(r.Name, r.Version, major), Major: major, LTS: lts, Provider: "Eclipse Temurin", Platform: platform, URL: address, SHA256: checksum, Channel: channel, Kind: releaseKind})
					kept++
				}
			}
			if len(rows) < javaPageSize || c.JavaRecent && (kept >= javaPageSize || page+1 >= javaRecentPages) {
				break
			}
		}
	}
	return out, nil
}

var javaPreviewName = regexp.MustCompile(`(?i)(?:^|[-+._])(?:ea|beta|alpha|rc|internal|snapshot)(?:$|[-+._0-9])`)
var java8Update = regexp.MustCompile(`(?i)(?:jdk)?8u([0-9]+)(?:-b([0-9]+))?`)
var javaSearchMajor = regexp.MustCompile(`^(?:java|jdk)?-?(1\.8|[0-9]{1,2}|8u)$`)
var javaDatedName = regexp.MustCompile(`^jdk(?:[0-9]+u)?-[0-9]{4}-[0-9]{2}-[0-9]{2}-`)

func javaDisplayVersion(name string, version javaVersionData, major int) string {
	text := version.OpenJDK
	if major == 8 {
		if m := java8Update.FindStringSubmatch(name); m != nil {
			text = "8u" + m[1]
		} else if version.Security > 0 {
			text = fmt.Sprintf("8u%d", version.Security)
		}
	} else if text != "" {
		text, _, _ = strings.Cut(text, "+")
	}
	if text == "" {
		text = strings.TrimPrefix(strings.TrimPrefix(name, "jdk"), "-")
	}
	label := fmt.Sprintf("Java %d", major)
	if major == 8 {
		label += " (1.8)"
	}
	return label + " · " + text
}

func catalogReleaseNewer(a, b CatalogRelease) bool {
	if a.Tool != "java" || b.Tool != "java" {
		return releaseNewer(a.Version, b.Version)
	}
	if a.Channel != b.Channel {
		return a.Channel == "stable"
	}
	if a.Major != b.Major {
		return a.Major > b.Major
	}
	version := func(r CatalogRelease) string {
		if r.javaSortVersion != "" {
			return r.javaSortVersion
		}
		if r.RuntimeVersion != "" {
			return r.RuntimeVersion
		}
		if m := java8Update.FindStringSubmatch(r.Version); m != nil {
			return "8.0." + m[1] + "+" + m[2]
		}
		// A date in a release ID is never a Java major/version number.
		if javaDatedName.MatchString(r.Version) {
			return strconv.Itoa(r.Major)
		}
		return r.Version
	}
	return releaseNewer(version(a), version(b))
}

// CatalogReleaseMatches accepts common Java family aliases without changing
// Version, which remains the exact upstream installation identity.
func CatalogReleaseMatches(r CatalogRelease, search string) bool {
	search = strings.ToLower(strings.TrimSpace(search))
	if r.Tool != "java" {
		return strings.Contains(strings.ToLower(r.Version), search)
	}
	haystack := strings.ToLower(strings.Join([]string{r.Version, r.RuntimeVersion, r.DisplayVersion, r.Provider}, " "))
	for _, token := range strings.Fields(search) {
		if token == "java" || token == "jdk" {
			continue
		}
		if m := javaSearchMajor.FindStringSubmatch(token); m != nil {
			text := m[1]
			if text == "1.8" || text == "8u" {
				text = "8"
			}
			major, _ := strconv.Atoi(text)
			if r.Major != major {
				return false
			}
			continue
		}
		if !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}
