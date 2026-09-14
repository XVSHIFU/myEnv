package backend

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"myenv/internal/config"
	"myenv/internal/runner"
)

func SDKEntry(directory, tool, platform string) string {
	name := tool
	if tool == "rust" {
		name = "rustc"
	}
	if platform == "windows-amd64" {
		name += ".exe"
	}
	return filepath.Join(directory, "sdks", tool, "bin", name)
}

func CheckSDKEntries(directory, tool, platform string) error {
	names := map[string][]string{"java": {"java", "javac", "jar"}, "go": {"go", "gofmt"}, "rust": {"rustc", "cargo", "rustdoc"}}
	for _, name := range names[tool] {
		if platform == "windows-amd64" {
			name += ".exe"
		}
		entry := filepath.Join(directory, "sdks", tool, "bin", name)
		info, err := os.Lstat(entry)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%s SDK companion is unavailable: %s", tool, name)
		}
	}
	return nil
}

var releaseNumbers = regexp.MustCompile(`[0-9]+`)

func releaseNewer(a, b string) bool {
	split := func(s string) []int {
		main, build, _ := strings.Cut(s, "+")
		nums := releaseNumbers.FindAllString(main, -1)
		out := make([]int, 5)
		for i, n := range nums {
			if i >= 4 {
				break
			}
			out[i], _ = strconv.Atoi(n)
		}
		if build != "" {
			out[4], _ = strconv.Atoi(build)
		}
		return out
	}
	aa, bb := split(a), split(b)
	for i := range aa {
		if aa[i] != bb[i] {
			return aa[i] > bb[i]
		}
	}
	return a > b
}

func (c Catalog) ResolveSDK(ctx context.Context, tool, selector, platform string) (config.RuntimeLock, error) {
	var empty config.RuntimeLock
	if tool == "rust" && config.IsPreview(tool, selector) {
		r, e := c.RustChannel(ctx, selector, platform)
		if e != nil {
			return empty, e
		}
		return config.RuntimeLock{Version: r.Version, RuntimeVersion: r.RuntimeVersion, Backend: "rust-official-v1", Evidence: "artifact", URL: r.URL, SHA256: r.SHA256}, nil
	}
	constraint, err := config.ParseConstraint(tool, selector)
	if err != nil {
		return empty, err
	}
	if tool == "java" {
		selector = config.NormalizeJavaSelector(selector)
		// A discovery-only page limit must never hide an explicit old SDK ID.
		c.JavaRecent = false
		nums := releaseNumbers.FindString(selector)
		c.JavaMajor, _ = strconv.Atoi(nums)
		c.IncludePreview = config.IsPreview(tool, selector)
	}
	rows, err := c.List(ctx, tool, platform)
	if err != nil {
		return empty, err
	}
	var chosen CatalogRelease
	for _, r := range rows {
		if r.Kind == "release_page" || (r.Channel != "stable" && !config.IsPreview(tool, selector)) || !constraint.Contains(r.Version) {
			continue
		}
		if chosen.Version == "" || catalogReleaseNewer(r, chosen) {
			chosen = r
		}
	}
	if chosen.Version == "" {
		return empty, fmt.Errorf("no stable %s release matches %q on %s", tool, selector, platform)
	}
	if chosen.SHA256 == "" {
		data, err := c.get(ctx, chosen.URL+".sha256", 4096)
		if err != nil {
			return empty, err
		}
		fields := strings.Fields(string(data))
		if len(fields) == 0 {
			return empty, fmt.Errorf("missing upstream checksum")
		}
		chosen.SHA256 = fields[0]
	}
	sum, err := hex.DecodeString(chosen.SHA256)
	if err != nil || len(sum) != 32 {
		return empty, fmt.Errorf("invalid upstream checksum")
	}
	return config.RuntimeLock{Version: chosen.Version, RuntimeVersion: chosen.RuntimeVersion, Backend: tool + "-official-v1", Evidence: "artifact", URL: chosen.URL, SHA256: strings.ToLower(chosen.SHA256)}, nil
}

func ValidateSDKOrigin(tool, address string) error {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || path.Clean(u.Path) != u.Path {
		return fmt.Errorf("invalid SDK source URL")
	}
	allowed := false
	switch tool {
	case "java":
		allowed = u.Host == "github.com" && strings.HasPrefix(u.Path, "/adoptium/temurin") && strings.Contains(u.Path, "-binaries/releases/download/")
	case "go":
		allowed = u.Host == "go.dev" && strings.HasPrefix(u.Path, "/dl/go")
	case "rust":
		allowed = u.Host == "static.rust-lang.org" && strings.HasPrefix(u.Path, "/dist/")
	}
	if !allowed {
		return fmt.Errorf("untrusted %s artifact origin", tool)
	}
	return nil
}

func SDKClient(base *http.Client) *http.Client {
	c := *base
	// SDK archives are substantially larger than metadata; cancellation remains explicit.
	c.Timeout = 30 * time.Minute
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many download redirects")
		}
		if req.URL.Scheme != "https" {
			return fmt.Errorf("SDK download redirected away from HTTPS")
		}
		switch req.URL.Host {
		case "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com", "go.dev", "dl.google.com", "static.rust-lang.org", "www.python.org":
			return nil
		}
		return fmt.Errorf("untrusted SDK download redirect host")
	}
	return &c
}

// PrepareSDK uses only verified archives inside an already tracked generation.
func PrepareSDK(ctx context.Context, archive, directory, tool, platform string, locked config.RuntimeLock) error {
	staging := filepath.Join(directory, "sdk-unpack-"+tool)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	var err error
	if strings.HasSuffix(locked.URL, ".zip") {
		err = ExtractZIP(archive, staging)
	} else if strings.HasSuffix(locked.URL, ".tar.gz") {
		err = ExtractTarGZ(archive, staging)
	} else {
		return fmt.Errorf("unsupported SDK archive format")
	}
	if err != nil {
		return err
	}
	children, err := os.ReadDir(staging)
	if err != nil {
		return err
	}
	if len(children) != 1 || !children[0].IsDir() {
		return fmt.Errorf("unexpected SDK archive root")
	}
	source := filepath.Join(staging, children[0].Name())
	destination := filepath.Join(directory, "sdks", tool)
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	if tool != "rust" {
		err = renameSDK(ctx, source, destination)
	} else {
		if err = os.Mkdir(destination, 0700); err != nil {
			return err
		}
		target := "x86_64-unknown-linux-gnu"
		if platform == "windows-amd64" {
			target = "x86_64-pc-windows-msvc"
		}
		// Official standalone component payloads; no install.sh and no host writes.
		for _, component := range []string{"rustc", "cargo", "rust-std-" + target} {
			root := filepath.Join(source, component)
			if _, err = os.Stat(root); err != nil {
				return err
			}
			err = filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
				if e != nil {
					return e
				}
				if e = ctx.Err(); e != nil {
					return e
				}
				rel, e := filepath.Rel(root, path)
				if e != nil {
					return e
				}
				if rel == "." {
					return nil
				}
				first := strings.Split(filepath.ToSlash(rel), "/")[0]
				if first != "bin" && first != "lib" && first != "libexec" {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				dest := filepath.Join(destination, rel)
				if d.IsDir() {
					return os.MkdirAll(dest, 0700)
				}
				if d.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("unsupported Rust component symbolic link")
				}
				// Payloads must not overwrite another component's files.
				if _, e = os.Lstat(dest); e == nil {
					return fmt.Errorf("duplicate Rust component file")
				}
				return os.Rename(path, dest)
			})
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}
	// Staging belongs to this newly registered generation; retaining it on error
	// is safe and allows ordinary operation recovery to reclaim the generation.
	entry := SDKEntry(directory, tool, platform)
	if err = CheckSDKEntries(directory, tool, platform); err != nil {
		return err
	}
	args := []string{"--version"}
	if tool == "java" {
		args = []string{"-version"}
	}
	if tool == "go" {
		args = []string{"version"}
	}
	info, err := os.Lstat(entry)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("SDK entry is missing or not regular")
	}
	output := &boundedOutput{limit: 8192}
	env, err := runner.Environment(os.Environ(), map[string]string{"GOTOOLCHAIN": "local"}, []string{filepath.Dir(entry)}, platform == "windows-amd64")
	if err != nil {
		return err
	}
	code, err := executeBackend(ctx, runner.Process{Executable: entry, Args: args, Directory: directory, Environment: env, Stdout: output, Stderr: output})
	if err != nil || code != 0 {
		return fmt.Errorf("%s verification failed (exit %d): %w", tool, code, err)
	}
	verified := false
	fields := strings.Fields(string(output.data))
	if tool == "go" {
		verified = len(fields) >= 3 && fields[0] == "go" && fields[1] == "version" && fields[2] == "go"+locked.Version
	}
	if tool == "rust" {
		want := locked.Version
		if locked.RuntimeVersion != "" {
			want = locked.RuntimeVersion
		}
		verified = len(fields) >= 2 && fields[0] == "rustc" && fields[1] == want
	}
	if tool == "java" {
		verified = javaVersionMatches(locked, string(output.data))
	}
	if !verified {
		return fmt.Errorf("%s version output differs from lock: %s", tool, output.data)
	}
	return os.RemoveAll(staging)
}

func javaVersionMatches(locked config.RuntimeLock, output string) bool {
	expected := strings.TrimPrefix(strings.TrimPrefix(locked.Version, "jdk"), "-")
	build := ""
	if locked.RuntimeVersion != "" {
		expected = locked.RuntimeVersion
	}
	expected = strings.Split(expected, "+")[0]
	if strings.HasPrefix(expected, "8u") {
		expected = "1.8.0_" + strings.TrimPrefix(expected, "8u")
	}
	if strings.HasPrefix(expected, "1.8.0_") {
		// Adoptium metadata includes JDK 8's -bNN build, while the quoted
		// java -version value omits it. Check that build on the runtime line.
		if index := strings.LastIndex(expected, "-b"); index >= 0 {
			if _, err := strconv.Atoi(expected[index+2:]); err == nil {
				build = expected
				expected = expected[:index]
			}
		}
	}
	if config.IsPreview("java", locked.Version) && locked.RuntimeVersion == "" {
		expected += "-ea"
	}
	return strings.Contains(output, "\""+expected+"\"") && (build == "" || strings.Contains(output, "(build "+build+")"))
}
