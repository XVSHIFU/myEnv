package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Package inspection reads declarations and installed metadata. It never runs
// install/update commands, project scripts, or package-manager dispatchers.
type InventoryPackage struct {
	Name      string `json:"name"`
	Requested string `json:"requested,omitempty"`
	Version   string `json:"version,omitempty"`
	Kind      string `json:"kind,omitempty"`
	State     string `json:"state"`
	Path      string `json:"path,omitempty"`
}
type PackageGroup struct {
	Ecosystem   string             `json:"ecosystem"`
	Scope       string             `json:"scope"`
	Root        string             `json:"root"`
	Manager     string             `json:"manager,omitempty"`
	Interpreter string             `json:"interpreter,omitempty"`
	State       string             `json:"state"`
	Problem     string             `json:"problem,omitempty"`
	Coverage    string             `json:"coverage,omitempty"`
	Packages    []InventoryPackage `json:"packages"`
	Truncated   bool               `json:"truncated,omitempty"`
}
type InventoryTool struct {
	Name        string `json:"name"`
	Ecosystem   string `json:"ecosystem"`
	Path        string `json:"path,omitempty"`
	Version     string `json:"version,omitempty"`
	State       string `json:"state"`
	Problem     string `json:"problem,omitempty"`
	Interpreter string `json:"interpreter,omitempty"`
}
type ProjectInventory struct {
	Directory string       `json:"directory"`
	Manifests []string     `json:"manifests"`
	Node      PackageGroup `json:"node"`
	Python    PackageGroup `json:"python"`
}

func (s *Service) InventoryForProject(ctx context.Context, deep bool, directory string) (Inventory, error) {
	result, err := s.Inventory(ctx, deep)
	if err != nil || directory == "" {
		return result, err
	}
	root, err := filepath.Abs(directory)
	if err != nil {
		return result, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return result, err
	}
	if !info.IsDir() {
		return result, fmt.Errorf("project inspection requires a directory")
	}
	managedPython, managedNode := "", ""
	projectService := *s
	projectService.Profile = false
	if status, err := projectService.Status(ctx, root); err == nil && sameInventoryPath(status.Project, root) && status.Generation != nil && status.Environment != "incomplete" {
		managedPython, managedNode = status.Generation.PythonExecutable, status.Generation.NodeExecutable
	}
	project := ProjectInventory{Directory: root, Manifests: []string{}, Node: inspectNodeProject(root), Python: inspectPythonProject(ctx, root, deep, managedPython)}
	project.Node.Interpreter = managedNode
	for _, name := range []string{"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "pyproject.toml", "uv.lock", "requirements.txt", ".python-version", ".node-version", ".nvmrc", "myenv.yaml"} {
		if info, err := os.Stat(filepath.Join(root, name)); err == nil && info.Mode().IsRegular() {
			project.Manifests = append(project.Manifests, name)
		}
	}
	result.Project = &project
	return result, ctx.Err()
}

func inspectInventoryTools(ctx context.Context, commands []ResolvedCommand, deep bool) ([]InventoryTool, []PackageGroup) {
	tools := []InventoryTool{}
	groups := []PackageGroup{}
	node, python := ResolvedCommand{}, ResolvedCommand{}
	for _, command := range commands {
		if command.Tool == "node" {
			node = command
		}
		if command.Tool == "python" {
			python = command
		}
	}
	for _, name := range []string{"npm", "pnpm"} {
		tool := InventoryTool{Name: name, Ecosystem: "node", State: "not_found"}
		entry, err := exec.LookPath(name)
		if err == nil && filepath.IsAbs(entry) {
			tool.Path, tool.State = entry, "unverified"
			tool.Problem = "dispatcher not executed; inspection never provisions package managers"
			for _, manifest := range nodeToolManifests(entry, name) {
				pkg, err := readNodeManifest(manifest)
				if err == nil && pkg.Name == name && pkg.Version != "" {
					tool.Version, tool.State, tool.Problem = pkg.Version, "available", ""
					break
				}
			}
		}
		if name == "npm" && tool.Version == "" && node.Path != "" {
			for _, manifest := range nodeToolManifests(node.Path, name) {
				pkg, err := readNodeManifest(manifest)
				if err == nil && pkg.Name == name && pkg.Version != "" {
					tool.Version, tool.State, tool.Problem = pkg.Version, "available", ""
					tool.Path = manifest
					break
				}
			}
		}
		tools = append(tools, tool)
	}
	uv := InventoryTool{Name: "uv", Ecosystem: "python", State: "not_found"}
	if entry, err := exec.LookPath("uv"); err == nil && filepath.IsAbs(entry) {
		uv.Path, uv.State = entry, "unverified"
		if deep {
			output, err := probeInventoryCommand(ctx, entry, []string{"--version"}, "", 8192)
			if err != nil {
				uv.Problem = err.Error()
			} else if strings.HasPrefix(output, "uv ") {
				uv.Version, uv.State = strings.TrimPrefix(output, "uv "), "available"
			} else {
				uv.Problem = "output does not identify uv"
			}
		}
	}
	tools = append(tools, uv)
	pip := InventoryTool{Name: "pip", Ecosystem: "python", State: "not_found"}
	if python.Path != "" && python.State == "available" && deep {
		group := inspectPythonPackages(ctx, python.Path, "interpreter", "")
		groups = append(groups, group)
		pip.Interpreter, pip.Path = python.Path, python.Path
		pip.State, pip.Problem = "unverified", group.Problem
		for _, pkg := range group.Packages {
			if strings.EqualFold(pkg.Name, "pip") {
				pip.Version, pip.State, pip.Problem = pkg.Version, "available", ""
			}
		}
		if group.State == "available" && pip.Version == "" {
			pip.State = "not_found"
		}
	}
	tools = append(tools, pip)
	if node.Path != "" {
		for _, root := range npmInventoryRoots(node.Path) {
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				groups = append(groups, inspectNodeGlobal(root))
			}
		}
	}
	return tools, groups
}

type nodeInventoryManifest struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	PackageManager       string            `json:"packageManager"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

func readInventoryFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("metadata is not a regular file")
	}
	const limit = 1024 * 1024
	if info.Size() > limit {
		return nil, fmt.Errorf("metadata exceeds 1 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if len(data) > limit {
		return nil, fmt.Errorf("metadata exceeds 1 MiB")
	}
	return data, err
}

func readNodeManifest(path string) (nodeInventoryManifest, error) {
	var result nodeInventoryManifest
	data, err := readInventoryFile(path)
	if err == nil {
		err = json.Unmarshal(data, &result)
	}
	return result, err
}

func validPackageName(name string) bool {
	if name == "" || strings.ContainsAny(name, "\\\x00") {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) > 2 || (len(parts) == 2 && !strings.HasPrefix(parts[0], "@")) {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return !filepath.IsAbs(name)
}

func inspectNodeProject(root string) PackageGroup {
	group := PackageGroup{Ecosystem: "node", Scope: "project", Root: root, State: "not_found", Packages: []InventoryPackage{}}
	manifest, err := readNodeManifest(filepath.Join(root, "package.json"))
	if err != nil {
		if !os.IsNotExist(err) {
			group.State, group.Problem = "unverified", err.Error()
		}
		return group
	}
	group.State, group.Manager = "available", manifest.PackageManager
	if group.Manager == "" {
		for _, lock := range []struct{ name, manager string }{{"pnpm-lock.yaml", "pnpm"}, {"package-lock.json", "npm"}, {"yarn.lock", "yarn"}} {
			if _, err := os.Stat(filepath.Join(root, lock.name)); err == nil {
				group.Manager = lock.manager
				break
			}
		}
	}
	for _, category := range []struct {
		kind   string
		values map[string]string
	}{{"dependency", manifest.Dependencies}, {"development", manifest.DevDependencies}, {"optional", manifest.OptionalDependencies}} {
		names := make([]string, 0, len(category.values))
		for name := range category.values {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if len(group.Packages) >= 1024 {
				group.Truncated = true
				break
			}
			pkg := InventoryPackage{Name: name, Requested: category.values[name], Kind: category.kind, State: "declared"}
			if !validPackageName(name) {
				pkg.State = "invalid"
			} else {
				path := filepath.Join(root, "node_modules", filepath.FromSlash(name), "package.json")
				installed, err := readNodeManifest(path)
				if err == nil && installed.Version != "" {
					pkg.Version, pkg.Path, pkg.State = installed.Version, filepath.Dir(path), "installed"
				}
			}
			group.Packages = append(group.Packages, pkg)
		}
	}
	return group
}

func nodeToolManifests(entry, name string) []string {
	dir := filepath.Dir(entry)
	paths := []string{filepath.Join(dir, "node_modules", name, "package.json"), filepath.Join(filepath.Dir(dir), "lib", "node_modules", name, "package.json")}
	if resolved, err := filepath.EvalSymlinks(entry); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(filepath.Dir(resolved)), "package.json"))
	}
	return paths
}

func npmInventoryRoots(node string) []string {
	// These are identified package roots, not a claim that every shell's custom
	// npm prefix is covered. Configuration is read as data; npm is not launched.
	roots := []string{}
	addPrefix := func(prefix string) {
		if filepath.IsAbs(prefix) {
			if runtime.GOOS == "windows" {
				roots = append(roots, filepath.Join(prefix, "node_modules"))
			} else {
				roots = append(roots, filepath.Join(prefix, "lib", "node_modules"))
			}
		}
	}
	addPrefix(os.Getenv("NPM_CONFIG_PREFIX"))
	if home, err := os.UserHomeDir(); err == nil {
		if data, err := readInventoryFile(filepath.Join(home, ".npmrc")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
				if ok && strings.EqualFold(strings.TrimSpace(key), "prefix") {
					value = strings.Trim(strings.TrimSpace(value), "\"'")
					if !strings.Contains(value, "${") {
						addPrefix(value)
					}
				}
			}
		}
	}
	if runtime.GOOS == "windows" {
		addPrefix(os.Getenv("APPDATA") + string(filepath.Separator) + "npm")
		roots = append(roots, filepath.Join(filepath.Dir(node), "node_modules"))
	} else {
		roots = append(roots, filepath.Join(filepath.Dir(filepath.Dir(node)), "lib", "node_modules"))
	}
	seen := map[string]bool{}
	result := []string{}
	for _, root := range roots {
		key := filepath.Clean(root)
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, root)
		}
	}
	return result
}

func inspectNodeGlobal(root string) PackageGroup {
	group := PackageGroup{Ecosystem: "node", Scope: "global_package_root", Root: root, Manager: "npm/pnpm", State: "available", Packages: []InventoryPackage{}}
	entries, truncated, err := inventoryDirectoryEntries(root, 1024)
	if err != nil {
		group.State, group.Problem = "unverified", err.Error()
		return group
	}
	group.Truncated = truncated
	appendPackage := func(path string) {
		if len(group.Packages) >= 1024 {
			group.Truncated = true
			return
		}
		pkg, err := readNodeManifest(filepath.Join(path, "package.json"))
		if err == nil && pkg.Name != "" {
			group.Packages = append(group.Packages, InventoryPackage{Name: pkg.Name, Version: pkg.Version, State: "installed", Path: path, Kind: "global"})
		}
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if strings.HasPrefix(entry.Name(), "@") {
			children, truncated, _ := inventoryDirectoryEntries(path, 1024-len(group.Packages))
			group.Truncated = group.Truncated || truncated
			for _, child := range children {
				appendPackage(filepath.Join(path, child.Name()))
			}
		} else {
			appendPackage(path)
		}
	}
	sort.Slice(group.Packages, func(i, j int) bool { return group.Packages[i].Name < group.Packages[j].Name })
	return group
}

func inventoryDirectoryEntries(root string, limit int) ([]os.DirEntry, bool, error) {
	if limit <= 0 {
		return nil, true, nil
	}
	f, err := os.Open(root)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	entries, err := f.ReadDir(limit + 1)
	if err == io.EOF {
		err = nil
	}
	truncated := len(entries) > limit
	if truncated {
		entries = entries[:limit]
	}
	return entries, truncated, err
}

func inspectPythonPackages(ctx context.Context, interpreter, scope, directory string) PackageGroup {
	group := PackageGroup{Ecosystem: "python", Scope: scope, Root: directory, Interpreter: interpreter, Manager: "pip/uv", State: "unverified", Packages: []InventoryPackage{}}
	// -I -S prevents PYTHONPATH, .pth files and sitecustomize from running during
	// inspection. Read only standard distribution metadata for this interpreter.
	// Before Python 3.14, -S reports the base prefix for a venv, so identify the
	// selected venv from its executable and pyvenv.cfg instead of borrowing base.
	script := `import sys,json,itertools,sysconfig,pathlib,importlib.metadata as m
entry=pathlib.Path(sys.executable)
prefix=entry.parent.parent if entry.parent.name.lower() in ('bin','scripts') else entry.parent
cfg=prefix/'pyvenv.cfg'
if cfg.is_file():
 roots=[str(prefix/'Lib'/'site-packages')] if sys.platform=='win32' else [str(prefix/'lib'/('python%d.%d'%sys.version_info[:2])/'site-packages')]
 text=cfg.open(encoding='utf-8',errors='replace').read(4096)
 if any(line.strip().lower().replace(' ','')=='include-system-site-packages=true' for line in text.splitlines()):
  paths=sysconfig.get_paths(vars={'base':sys.base_prefix,'platbase':sys.base_exec_prefix}); roots+=list(dict.fromkeys((paths['purelib'],paths['platlib'])))
else:
 prefix=pathlib.Path(sys.prefix); paths=sysconfig.get_paths(); roots=list(dict.fromkeys((paths['purelib'],paths['platlib'])))
d=list(itertools.islice(m.distributions(path=list(dict.fromkeys(roots))),1025)); truncated=len(d)>1024
d=sorted(d[:1024],key=lambda x:(x.metadata.get('Name') or '').lower())
print(json.dumps({'root':str(prefix),'packages':[{'name':x.metadata.get('Name') or '', 'version':x.version,'state':'installed','path':str(x.locate_file(''))} for x in d],'truncated':truncated}))`
	output, err := probeInventoryCommand(ctx, interpreter, []string{"-I", "-S", "-c", script}, directory, 256*1024)
	if err != nil {
		group.Problem = err.Error()
		return group
	}
	var decoded struct {
		Root      string             `json:"root"`
		Packages  []InventoryPackage `json:"packages"`
		Truncated bool               `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		group.Problem = "Python metadata output could not be read"
		return group
	}
	group.Root, group.Packages, group.Truncated, group.State = decoded.Root, decoded.Packages, decoded.Truncated, "available"
	group.Coverage = "Standard interpreter site-packages inspected in isolated mode; user site and PYTHONPATH are excluded."
	return group
}

func inspectPythonProject(ctx context.Context, root string, deep bool, managedInterpreters ...string) PackageGroup {
	group := PackageGroup{Ecosystem: "python", Scope: "project", Root: root, State: "not_found", Packages: []InventoryPackage{}}
	if data, err := readInventoryFile(filepath.Join(root, "pyproject.toml")); err == nil {
		var project struct {
			Project struct {
				Dependencies []string `toml:"dependencies"`
			} `toml:"project"`
		}
		if err := toml.Unmarshal(data, &project); err != nil {
			group.State, group.Problem = "unverified", "pyproject.toml could not be read"
		} else {
			group.State = "declared"
			for _, dep := range project.Project.Dependencies {
				if len(group.Packages) >= 1024 {
					group.Truncated = true
					break
				}
				group.Packages = append(group.Packages, InventoryPackage{Name: pythonDependencyName(dep), Requested: dep, Kind: "dependency", State: "declared"})
			}
		}
	}
	if group.State == "not_found" {
		if data, err := readInventoryFile(filepath.Join(root, "requirements.txt")); err == nil {
			group.State = "declared"
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if strings.HasPrefix(line, "-") || strings.HasSuffix(line, "\\") {
					group.Coverage = "Only direct requirement lines are displayed; includes and command options are not executed."
					continue
				}
				if len(group.Packages) >= 1024 {
					group.Truncated = true
					break
				}
				group.Packages = append(group.Packages, InventoryPackage{Name: pythonDependencyName(line), Requested: line, Kind: "dependency", State: "declared"})
			}
		}
	}
	candidates := append([]string{}, managedInterpreters...)
	for _, name := range []string{".venv", "venv"} {
		base := filepath.Join(root, name)
		interpreter := filepath.Join(base, "bin", "python")
		if runtime.GOOS == "windows" {
			interpreter = filepath.Join(base, "Scripts", "python.exe")
		}
		if _, err := os.Stat(filepath.Join(base, "pyvenv.cfg")); err != nil {
			continue
		}
		candidates = append(candidates, interpreter)
	}
	for _, interpreter := range candidates {
		if interpreter == "" {
			continue
		}
		if info, err := os.Stat(interpreter); err != nil || !info.Mode().IsRegular() {
			continue
		}
		group.Interpreter = interpreter
		if deep {
			installed := inspectPythonPackages(ctx, interpreter, "project", root)
			if installed.State == "available" {
				for _, declared := range group.Packages {
					found := false
					for i := range installed.Packages {
						if normalizePythonPackage(installed.Packages[i].Name) == normalizePythonPackage(declared.Name) {
							installed.Packages[i].Requested, installed.Packages[i].Kind = declared.Requested, declared.Kind
							found = true
							break
						}
					}
					if !found {
						installed.Packages = append(installed.Packages, declared)
					}
				}
				return installed
			}
			group.Problem = installed.Problem
		}
		if group.State == "not_found" {
			group.State = "unverified"
		}
		return group
	}
	if group.State == "declared" {
		group.Problem = "No .venv/venv interpreter found; declaration only, machine Python is not assumed to be the project's environment."
	}
	return group
}

func pythonDependencyName(value string) string {
	value = strings.TrimSpace(value)
	end := strings.IndexAny(value, "[<>=!~; @\t")
	if end >= 0 {
		return value[:end]
	}
	return value
}
func normalizePythonPackage(value string) string {
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(value))
}
