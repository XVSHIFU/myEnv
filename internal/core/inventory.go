package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
)

type Installation struct {
	ID      string   `json:"id"`
	Tool    string   `json:"tool"`
	Path    string   `json:"path"`
	Source  string   `json:"discovery_source"`
	Owner   string   `json:"owner"`
	Manager string   `json:"manager"`
	State   string   `json:"state"`
	Version string   `json:"version_output,omitempty"`
	Problem string   `json:"problem,omitempty"`
	Actions []string `json:"actions"`
}
type Inventory struct {
	Compilers     []CompilerCheck `json:"compiler_prerequisites,omitempty"`
	Installations []Installation  `json:"installations"`
	Coverage      string          `json:"coverage"`
	Warnings      []string        `json:"warnings"`
}
type inventoryCandidate struct{ tool, path, source string }

func (s *Service) Inventory(ctx context.Context, deep bool, extraPaths ...string) (Inventory, error) {
	result := Inventory{Installations: []Installation{}, Warnings: []string{}, Coverage: "PATH, explicit runtime homes, common user/system locations, Python/Conda registrations and current myenv user profile; arbitrary custom directories and other users are not exhaustively scanned"}
	candidates := []inventoryCandidate{}
	names := map[string][]string{"python": {"python", "python3"}, "node": {"node"}, "java": {"java"}, "go": {"go"}, "rust": {"rustc"}}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	addBin := func(dir, source string) {
		if dir == "" || !filepath.IsAbs(dir) {
			return
		}
		for tool, entries := range names {
			for _, entry := range entries {
				p := filepath.Join(dir, entry+suffix)
				if _, err := os.Lstat(p); err == nil {
					candidates = append(candidates, inventoryCandidate{tool, p, source})
				}
			}
		}
	}
	for _, extra := range extraPaths {
		absolute, e := filepath.Abs(extra)
		if e != nil {
			return result, e
		}
		info, e := os.Stat(absolute)
		if e != nil {
			result.Warnings = append(result.Warnings, "explicit path: "+e.Error())
			continue
		}
		if info.IsDir() {
			addBin(absolute, "explicit path")
			addBin(filepath.Join(absolute, "bin"), "explicit path")
		} else {
			name := strings.TrimSuffix(strings.ToLower(filepath.Base(absolute)), ".exe")
			for tool, entries := range names {
				for _, entry := range entries {
					if name == entry {
						candidates = append(candidates, inventoryCandidate{tool, absolute, "explicit path"})
					}
				}
			}
		}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		addBin(dir, "PATH")
	}
	for tool, key := range map[string]string{"java": "JAVA_HOME", "go": "GOROOT", "python": "CONDA_PREFIX"} {
		if home := os.Getenv(key); home != "" && filepath.IsAbs(home) {
			bin := filepath.Join(home, "bin")
			if tool == "python" && runtime.GOOS == "windows" {
				bin = home
			}
			candidates = append(candidates, inventoryCandidate{tool, filepath.Join(bin, names[tool][0]+suffix), key})
		}
	}
	home, _ := os.UserHomeDir()
	for _, dir := range []string{filepath.Join(home, ".cargo", "bin"), filepath.Join(home, ".local", "bin"), "/usr/bin", "/usr/local/bin", "/usr/local/go/bin"} {
		addBin(dir, "standard location")
	}
	for _, dir := range []string{filepath.Join(os.Getenv("ProgramFiles"), "Go", "bin"), filepath.Join(os.Getenv("ProgramFiles"), "nodejs")} {
		addBin(dir, "standard Windows location")
	}
	// Enumerate one bounded level of known manager directories, never the disk.
	roots := map[string]string{
		filepath.Join(home, ".rustup", "toolchains"):                   "rustup toolchains",
		filepath.Join(home, ".local", "share", "uv", "python"):         "uv managed directory",
		filepath.Join(os.Getenv("LOCALAPPDATA"), "uv", "python"):       "uv managed directory",
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python"): "Python user install location",
		filepath.Join(os.Getenv("ProgramFiles"), "Java"):               "Java install location",
		filepath.Join(os.Getenv("ProgramFiles"), "Eclipse Adoptium"):   "Temurin install location",
		filepath.Join(home, ".nvm", "versions", "node"):                "nvm managed directory",
		filepath.Join(home, ".volta", "tools", "image", "node"):        "Volta managed directory",
		filepath.Join(home, ".sdkman", "candidates", "java"):           "SDKMAN managed directory",
		filepath.Join(home, ".gvm", "gos"):                             "gvm managed directory",
		"/usr/lib/jvm":                                                 "Java install location",
	}
	if nvmHome := os.Getenv("NVM_HOME"); filepath.IsAbs(nvmHome) {
		roots[nvmHome] = "nvm managed directory"
	}
	for _, name := range []string{"python", "nodejs", "java", "golang", "rust"} {
		roots[filepath.Join(home, ".asdf", "installs", name)] = "asdf managed directory"
	}
	for directory, source := range roots {
		if !filepath.IsAbs(directory) {
			continue
		}
		f, e := os.Open(directory)
		if e != nil {
			if !os.IsNotExist(e) {
				result.Warnings = append(result.Warnings, e.Error())
			}
			continue
		}
		children, e := f.ReadDir(4097)
		f.Close()
		if len(children) > 4096 {
			result.Warnings = append(result.Warnings, "directory scan truncated: "+directory)
			children = children[:4096]
		}
		for _, child := range children {
			if child.IsDir() {
				base := filepath.Join(directory, child.Name())
				addBin(base, source)
				addBin(filepath.Join(base, "bin"), source)
			}
		}
	}
	// Conda records are read as data; Conda/Anaconda is never installed here.
	if data, err := config.ReadInput(filepath.Join(home, ".conda", "environments.txt")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !filepath.IsAbs(line) {
				continue
			}
			entry := filepath.Join(line, "bin", "python")
			if runtime.GOOS == "windows" {
				entry = filepath.Join(line, "python.exe")
			}
			candidates = append(candidates, inventoryCandidate{"python", entry, "Conda environment registry"})
		}
	}
	registered, warnings := registeredInventory()
	candidates = append(candidates, registered...)
	result.Warnings = append(result.Warnings, warnings...)
	managed := map[string]bool{}
	profile := *s
	profile.Profile = true
	status, err := profile.Status(ctx, "")
	if err == nil && status.Generation != nil {
		platform, e := runner.Platform()
		if e != nil {
			return result, e
		}
		snapshot, e := config.ReadSnapshot(status.Generation.Directory, status.Project)
		if e == nil {
			for tool := range snapshot.Tools {
				entry := status.Generation.NodeExecutable
				if tool == "python" {
					entry = status.Generation.PythonExecutable
				}
				if tool == "java" || tool == "go" || tool == "rust" {
					entry = backend.SDKEntry(status.Generation.Directory, tool, platform)
				}
				if entry != "" {
					candidates = append(candidates, inventoryCandidate{tool, entry, "myenv user profile"})
					managed[entry] = true
				}
			}
		}
	} else if err != nil && !os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, "myenv user profile: "+err.Error())
	}
	discoveries := map[string][]string{}
	for _, candidate := range candidates {
		k := filepath.Clean(candidate.path)
		if runtime.GOOS == "windows" {
			k = strings.ToLower(k)
		}
		present := false
		for _, source := range discoveries[k] {
			present = present || source == candidate.source
		}
		if !present {
			discoveries[k] = append(discoveries[k], candidate.source)
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		path := filepath.Clean(candidate.path)
		key := path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		row := Installation{ID: fmt.Sprintf("%x", sha256.Sum256([]byte(key)))[:16], Tool: candidate.tool, Path: path, Source: strings.Join(discoveries[key], "; "), Owner: "external", Manager: "unknown", State: "unverified", Actions: []string{"inspect"}}
		for _, manager := range []string{"nvm", "Volta", "SDKMAN", "gvm", "asdf"} {
			if strings.Contains(row.Source, manager+" managed") {
				row.Manager = manager
			}
		}
		if strings.Contains(row.Source, "uv managed") {
			row.Manager = "uv"
		}
		if strings.Contains(row.Source, "rustup") {
			row.Manager = "rustup"
		}
		if candidate.tool == "python" {
			for _, base := range []string{filepath.Dir(path), filepath.Dir(filepath.Dir(path))} {
				if info, e := os.Stat(filepath.Join(base, "conda-meta")); e == nil && info.IsDir() {
					row.Manager = "conda"
					break
				}
			}
		}
		if managed[path] {
			row.Owner = "myenv"
			row.Manager = "myenv"
			row.Actions = []string{"inspect", "upgrade", "repair", "remove"}
		}
		info, e := os.Stat(path)
		if strings.Contains(strings.ToLower(path), "\\microsoft\\windowsapps\\") {
			row.State = "alias"
			row.Problem = "Windows application execution alias; runtime not verified"
		} else if e != nil {
			row.State = "broken"
			row.Problem = e.Error()
		} else if !info.Mode().IsRegular() {
			row.State = "broken"
			row.Problem = "entry is not a regular file"
		} else if candidate.tool == "rust" && rustupShim(path) {
			row.State = "shim"
			row.Manager = "rustup"
			row.Problem = "rustup dispatcher; inspect its installed toolchains rather than triggering automatic installation"
		} else if deep {
			// WindowsApps Python aliases may open the Store; never execute them during discovery.
			if strings.Contains(strings.ToLower(path), "\\microsoft\\windowsapps\\") {
				row.State = "alias"
				row.Problem = "Windows application execution alias; runtime not verified"
			} else {
				args := []string{"--version"}
				if row.Tool == "java" {
					args = []string{"-version"}
				}
				if row.Tool == "go" {
					args = []string{"version"}
				}
				probe, cancel := context.WithTimeout(ctx, 5*time.Second)
				command := exec.CommandContext(probe, path, args...)
				command.Env, _ = runner.Environment(os.Environ(), map[string]string{"GOTOOLCHAIN": "local", "GOWORK": "off"}, nil, runtime.GOOS == "windows")
				command.WaitDelay = time.Second
				output := &inventoryOutput{}
				command.Stdout = output
				command.Stderr = output
				e = command.Run()
				cancel()
				row.Version = strings.TrimSpace(output.text.String())
				if e != nil {
					row.State = "broken"
					row.Problem = e.Error()
				} else if !inventoryVersionMatches(row.Tool, row.Version) {
					row.State = "unverified"
					row.Problem = "version output does not identify the expected runtime"
				} else {
					row.State = "available"
				}
			}
		}
		result.Installations = append(result.Installations, row)
	}
	sort.Slice(result.Installations, func(i, j int) bool {
		a, b := result.Installations[i], result.Installations[j]
		if a.Tool != b.Tool {
			return a.Tool < b.Tool
		}
		return a.Path < b.Path
	})
	return result, nil
}

type inventoryOutput struct{ text strings.Builder }

func (b *inventoryOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 8192 - b.text.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		b.text.Write(p)
	}
	return n, nil
}

func inventoryVersionMatches(tool, output string) bool {
	patterns := map[string]string{"node": `(?m)^v[0-9]+\.`, "python": `(?m)^Python [0-9]+\.`, "go": `(?m)^go version go[0-9]`, "rust": `(?m)^rustc [0-9]+\.`, "java": `(?m)^(?:openjdk|java) (?:version )?"?[0-9]`}
	pattern, ok := patterns[tool]
	return ok && regexp.MustCompile(pattern).MatchString(output)
}

func rustupShim(entry string) bool {
	name := "rustup"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	a, e := os.Stat(entry)
	if e != nil {
		return false
	}
	b, e := os.Stat(filepath.Join(filepath.Dir(entry), name))
	return e == nil && os.SameFile(a, b)
}
