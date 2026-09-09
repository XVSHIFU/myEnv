package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"myenv/internal/config"
	"myenv/internal/runner"
)

type ExternalPlan struct {
	Environment  map[string]string `json:"environment,omitempty"`
	Installation Installation      `json:"installation"`
	Manager      string            `json:"manager_executable"`
	Args         []string          `json:"args"`
	Directory    string            `json:"directory"`
	Notice       string            `json:"notice"`
}

type managerOutput struct {
	data     []byte
	overflow bool
}

func (b *managerOutput) Write(p []byte) (int, error) {
	n := len(p)
	room := (4 << 20) - len(b.data)
	if n > room {
		b.overflow = true
		p = p[:room]
	}
	b.data = append(b.data, p...)
	return n, nil
}

func managerRead(ctx context.Context, exe, dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out := &managerOutput{}
	diagnostic := &managerOutput{}
	code, e := runner.Execute(ctx, runner.Process{Executable: exe, Args: args, Directory: dir, Environment: os.Environ(), Stdout: out, Stderr: diagnostic})
	if e != nil {
		return nil, e
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if code != 0 {
		return nil, fmt.Errorf("manager query failed (%d): %s", code, diagnostic.data)
	}
	if out.overflow {
		return nil, fmt.Errorf("manager output exceeds 4 MiB")
	}
	return out.data, nil
}

func sameInstallPath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// BuildExternalPlan proves membership with the selected original manager.
// Discovery alone never grants permission to recursively delete directories.
func BuildExternalPlan(ctx context.Context, row Installation, manager, action string) (ExternalPlan, error) {
	return buildExternalPlan(ctx, row, manager, action, managerRead)
}

func buildExternalPlan(ctx context.Context, row Installation, manager, action string, read func(context.Context, string, string, ...string) ([]byte, error)) (ExternalPlan, error) {
	plan := ExternalPlan{Installation: row, Manager: manager, Notice: "由原管理器直接修改外部安装；不属于 myenv 环境代回滚。请先关闭使用该环境的程序。"}
	if row.Owner != "external" || (action != "upgrade" && action != "repair" && action != "remove") {
		return plan, fmt.Errorf("requires an external installation and upgrade, repair or remove")
	}
	checked := row.Path
	if resolved, e := config.ResolveExistingPath(row.Path); e == nil {
		checked = resolved
	} else if !os.IsNotExist(e) {
		return plan, e
	}
	checked = strings.ToLower(filepath.ToSlash(checked))
	if strings.Contains(checked, "/.myenv/") || strings.Contains(checked, "/myenv/runtimes/") {
		return plan, fmt.Errorf("myenv generation/shared runtime cannot be modified through an external manager")
	}
	if !filepath.IsAbs(manager) {
		return plan, fmt.Errorf("--manager requires the absolute original manager executable")
	}
	info, e := os.Lstat(manager)
	if e != nil {
		return plan, e
	}
	if !info.Mode().IsRegular() {
		return plan, fmt.Errorf("manager must be a regular executable, not a script alias or symlink")
	}
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(manager)), ".exe")
	if name != "uv" && name != "rustup" && name != "conda" && name != "_conda" {
		return plan, fmt.Errorf("supported external adapters: uv, rustup, conda; unknown installers remain read-only")
	}
	plan.Directory = filepath.Dir(manager)
	version, e := read(ctx, manager, plan.Directory, "--version")
	if e != nil {
		return plan, e
	}
	identity := name
	if identity == "_conda" {
		identity = "conda"
	}
	if !strings.HasPrefix(strings.TrimSpace(string(version)), identity+" ") {
		return plan, fmt.Errorf("manager identity does not match executable")
	}
	switch identity {
	case "rustup":
		if row.Tool != "rust" {
			return plan, fmt.Errorf("rustup manages only Rust")
		}
		root := filepath.Dir(filepath.Dir(row.Path))
		name := filepath.Base(root)
		if !regexp.MustCompile(`^(?:stable|beta|nightly|[0-9]+\.[0-9]+\.[0-9]+)(?:-[a-zA-Z0-9_.]+)+$`).MatchString(name) {
			return plan, fmt.Errorf("custom/linked rustup toolchains are not managed by this adapter")
		}
		data, e := read(ctx, manager, plan.Directory, "toolchain", "list", "-v")
		if e != nil {
			return plan, e
		}
		member := false
		for _, line := range strings.Split(string(data), "\n") {
			f := strings.Fields(line)
			if len(f) > 1 && f[0] == name && strings.HasSuffix(strings.TrimSpace(line), root) {
				member = true
			}
		}
		if !member {
			return plan, fmt.Errorf("rustup did not confirm this installation path")
		}
		if action == "repair" {
			return plan, fmt.Errorf("rustup has no atomic reinstall command; use a fresh myenv Rust installation instead of removing the existing toolchain first")
		}
		if action == "remove" {
			plan.Args = []string{"toolchain", "uninstall", name}
		} else {
			plan.Args = []string{"update", name, "--no-self-update"}
		}
	case "uv":
		if row.Tool != "python" {
			return plan, fmt.Errorf("uv adapter manages only Python runtimes")
		}
		root := filepath.Dir(row.Path)
		for i := 0; i < 3 && !strings.HasPrefix(filepath.Base(root), "cpython-"); i++ {
			root = filepath.Dir(root)
		}
		key := filepath.Base(root)
		if !regexp.MustCompile(`^cpython-[0-9][a-zA-Z0-9_.+-]+$`).MatchString(key) {
			return plan, fmt.Errorf("not a recognized uv managed runtime directory")
		}
		data, e := read(ctx, manager, plan.Directory, "python", "list", "--only-installed", "--managed-python", "--output-format", "json", "--no-config", "--offline", "--no-python-downloads")
		if e != nil {
			return plan, e
		}
		var rows []struct{ Key, Path string }
		if e = json.Unmarshal(data, &rows); e != nil {
			return plan, e
		}
		member := false
		for _, r := range rows {
			p := r.Path
			if !filepath.IsAbs(p) {
				p = filepath.Join(plan.Directory, p)
			}
			a, ea := os.Stat(p)
			b, eb := os.Stat(row.Path)
			registeredRoot := filepath.Dir(p)
			for i := 0; i < 3 && !strings.HasPrefix(filepath.Base(registeredRoot), "cpython-"); i++ {
				registeredRoot = filepath.Dir(registeredRoot)
			}
			ra, re := os.Stat(filepath.Dir(registeredRoot))
			rb, be := os.Stat(filepath.Dir(root))
			if ea == nil && eb == nil && re == nil && be == nil && os.SameFile(a, b) && os.SameFile(ra, rb) && regexp.MustCompile(`^cpython-[0-9]+\.[0-9]+\.[0-9]+[a-zA-Z0-9_.+-]*$`).MatchString(r.Key) {
				member = true
				key = r.Key
			}
		}
		if !member {
			return plan, fmt.Errorf("uv did not confirm this installed interpreter; custom UV_PYTHON_INSTALL_DIR must be configured in the invoking shell")
		}
		switch action {
		case "remove":
			plan.Args = []string{"python", "uninstall", key, "--install-dir", filepath.Dir(root), "--no-config"}
		case "repair":
			plan.Args = []string{"python", "install", key, "--reinstall", "--install-dir", filepath.Dir(root), "--no-bin", "--no-registry", "--no-config"}
		case "upgrade":
			parts := regexp.MustCompile(`^cpython-([0-9]+\.[0-9]+)\.`).FindStringSubmatch(key)
			if len(parts) != 2 {
				return plan, fmt.Errorf("cannot identify uv minor version")
			}
			if _, e = read(ctx, manager, plan.Directory, "python", "upgrade", "--help"); e != nil {
				return plan, e
			}
			plan.Args = []string{"python", "upgrade", parts[1], "--install-dir", filepath.Dir(root), "--no-config"}
			plan.Notice += " uv 的补丁升级功能处于预览阶段；该 minor 的安装及关联虚拟环境可能受影响。"
			plan.Environment = map[string]string{"UV_PYTHON_INSTALL_BIN": "0", "UV_PYTHON_INSTALL_REGISTRY": "0"}
		}
	case "conda":
		if row.Tool != "python" {
			return plan, fmt.Errorf("Conda adapter currently manages Python environments")
		}
		root := filepath.Dir(row.Path)
		if runtime.GOOS != "windows" {
			root = filepath.Dir(root)
		}
		if _, e = os.Stat(filepath.Join(root, "conda-meta", "history")); e != nil {
			return plan, fmt.Errorf("Conda history is missing: %w", e)
		}
		data, e := read(ctx, manager, plan.Directory, "info", "--json")
		if e != nil {
			return plan, e
		}
		var metadata struct {
			RootPrefix string `json:"root_prefix"`
		}
		if e = json.Unmarshal(data, &metadata); e != nil {
			return plan, e
		}
		if metadata.RootPrefix == "" {
			return plan, fmt.Errorf("Conda root is unknown")
		}
		if sameInstallPath(root, metadata.RootPrefix) {
			return plan, fmt.Errorf("Conda base environment is protected; use Conda directly to change its own runtime")
		}
		data, e = read(ctx, manager, plan.Directory, "env", "list", "--json")
		if e != nil {
			return plan, e
		}
		var list struct{ Envs []string }
		if e = json.Unmarshal(data, &list); e != nil {
			return plan, e
		}
		member := false
		for _, p := range list.Envs {
			if sameInstallPath(p, root) {
				member = true
			}
		}
		if !member {
			return plan, fmt.Errorf("Conda did not confirm this environment prefix")
		}
		switch action {
		case "upgrade":
			plan.Args = []string{"update", "--prefix", root, "python", "--yes"}
		case "repair":
			data, e = read(ctx, manager, plan.Directory, "list", "--prefix", root, "--json", "^python$")
			if e != nil {
				return plan, e
			}
			var packages []struct{ Name, Version, Build string }
			if e = json.Unmarshal(data, &packages); e != nil {
				return plan, e
			}
			if len(packages) != 1 || packages[0].Name != "python" || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[ab]|rc)?[0-9]*$`).MatchString(packages[0].Version) {
				return plan, fmt.Errorf("Conda Python version record is ambiguous")
			}
			spec := "python=" + packages[0].Version
			if packages[0].Build != "" {
				if !regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`).MatchString(packages[0].Build) {
					return plan, fmt.Errorf("invalid Conda build identifier")
				}
				spec += "=" + packages[0].Build
			}
			plan.Args = []string{"install", "--prefix", root, spec, "--force-reinstall", "--freeze-installed", "--yes"}
		case "remove":
			plan.Args = []string{"env", "remove", "--prefix", root, "--yes"}
			plan.Notice += " 删除操作会移除整个 Conda 环境及其中的包，不只移除 Python。"
		}
	}
	return plan, nil
}

func ApplyExternalPlan(ctx context.Context, plan ExternalPlan, in io.Reader, out, diagnostic io.Writer) error {
	env, e := runner.Environment(os.Environ(), plan.Environment, nil, runtime.GOOS == "windows")
	if e != nil {
		return e
	}
	code, e := runner.Execute(ctx, runner.Process{Executable: plan.Manager, Args: plan.Args, Directory: plan.Directory, Environment: env, Stdin: in, Stdout: out, Stderr: diagnostic})
	if e != nil {
		return e
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if code != 0 {
		return fmt.Errorf("external manager exited %d; inspect the original manager state before retrying", code)
	}
	return nil
}
