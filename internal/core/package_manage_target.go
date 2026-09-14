package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"myenv/internal/config"
	"myenv/internal/runner"
)

type packageManager struct{ Name, Executable, Script, Manifest, Version, Receipt string }

func (s *Service) packageTarget(ctx context.Context, target PackageTarget) (packageManager, PackageGroup, PackageTarget, error) {
	var manager packageManager
	var group PackageGroup
	if target.Ecosystem != "node" && target.Ecosystem != "python" {
		return manager, group, target, fmt.Errorf("本版支持 Node.js 和 Python 包管理")
	}
	if target.Scope == "tool" {
		var err error
		target, err = packageToolTarget(ctx, target)
		if err != nil {
			return manager, group, target, err
		}
	}
	if target.Scope != "project" && target.Scope != "interpreter" && target.Scope != "global_package_root" && target.Scope != "standalone_tool" {
		return manager, group, target, fmt.Errorf("请明确选择项目、全局包位置或 Python 解释器")
	}
	if target.Scope == "standalone_tool" {
		return standaloneUVTarget(ctx, target)
	}
	if target.Root == "" || !filepath.IsAbs(target.Root) {
		return manager, group, target, fmt.Errorf("请提供已检查环境的绝对路径")
	}
	root, err := config.ResolveExistingPath(target.Root)
	if err != nil {
		return manager, group, target, err
	}
	target.Root = root
	if protectedManagedPackagePath(root) {
		return manager, group, target, fmt.Errorf("myEnv 活动环境不能原地修改；请编辑项目依赖并预览同步，以保留旧环境和回退能力")
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return manager, group, target, fmt.Errorf("所选环境目录不存在")
	}
	if filepath.Dir(root) == root {
		return manager, group, target, fmt.Errorf("不能将磁盘根目录作为包管理目标")
	}
	if target.Directory == "" && target.Scope == "project" {
		target.Directory = root
	}
	if target.Directory != "" {
		target.Directory, err = config.ResolveExistingPath(target.Directory)
		if err != nil {
			return manager, group, target, err
		}
	}
	if target.Ecosystem == "node" {
		if target.Scope != "project" && target.Scope != "global_package_root" {
			return manager, group, target, fmt.Errorf("Node.js 包必须选择项目或全局包安装位置")
		}
		group = inspectPackageTarget(ctx, target)
		if group.State != "available" {
			return manager, group, target, fmt.Errorf("无法核验所选包位置：%s", group.Problem)
		}
		if target.Scope == "project" {
			if !sameInventoryPath(target.Directory, target.Root) {
				return manager, group, target, fmt.Errorf("Node.js 项目路径与安装位置不一致")
			}
			if err := checkPackageWorkspace(target.Root); err != nil {
				return manager, group, target, err
			}
			if target.Manager == "" {
				target.Manager = strings.Split(group.Manager, "@")[0]
			}
			if target.Manager == "" {
				target.Manager = "npm"
			}
			declaredManager := strings.Split(group.Manager, "@")[0]
			if declaredManager != "" && declaredManager != strings.Split(target.Manager, "@")[0] {
				return manager, group, target, fmt.Errorf("项目使用 %s，不能通过另一包管理器改写其依赖锁", group.Manager)
			}
		} else {
			if filepath.Base(target.Root) != "node_modules" {
				return manager, group, target, fmt.Errorf("全局包位置必须是核验过的 node_modules 目录")
			}
			if target.Manager == "" || target.Manager == "npm/pnpm" {
				target.Manager = "npm"
			}
			if target.Manager != "npm" {
				return manager, group, target, fmt.Errorf("此全局位置按 npm prefix 管理；pnpm 专用全局仓库请使用其原终端")
			}
			if _, err := nodeGlobalPrefix(target.Root); err != nil {
				return manager, group, target, err
			}
		}
		manager, err = resolveNodePackageManager(ctx, target)
		if err != nil {
			return manager, group, target, err
		}
		if target.Scope == "project" {
			if _, version, ok := strings.Cut(group.Manager, "@"); ok && strings.Split(version, "+")[0] != manager.Version {
				return manager, group, target, fmt.Errorf("项目指定 %s，但当前已安装 %s %s；请通过原管理器对齐版本后重试", group.Manager, manager.Name, manager.Version)
			}
		}
		target.Manager, target.ManagerPath, target.Interpreter = manager.Name, manager.Manifest, manager.Executable
		if target.Scope == "global_package_root" {
			prefix, _ := nodeGlobalPrefix(target.Root)
			probe := PackageCommand{Executable: manager.Executable, Args: []string{manager.Script, "root", "--global", "--prefix", prefix}, Directory: prefix}
			output, e := readPackageCommand(ctx, target, manager, probe)
			if e != nil || !sameInventoryPath(strings.TrimSpace(output), target.Root) {
				return manager, group, target, fmt.Errorf("npm 未确认所选全局安装位置；请刷新检查")
			}
		}
	} else {
		if target.Interpreter == "" || !filepath.IsAbs(target.Interpreter) {
			return manager, group, target, fmt.Errorf("请选择此环境实际使用的 Python 解释器")
		}
		if protectedManagedPackagePath(target.Interpreter) || protectedManagedPackagePath(target.Root) {
			return manager, group, target, fmt.Errorf("myEnv 活动 Python 环境不能原地修改；请编辑项目依赖并预览同步，以保留旧环境和回退能力")
		}
		group = inspectPackageTarget(ctx, target)
		if group.State != "available" {
			return manager, group, target, fmt.Errorf("Python 解释器未确认所选安装位置：%s", group.Problem)
		}
		if !sameInventoryPath(group.Root, target.Root) {
			return manager, group, target, fmt.Errorf("Python 解释器属于 %s，与所选环境 %s 不一致", group.Root, target.Root)
		}
		pip := findManagedPackage(group, "pip")
		if pip.Name == "" {
			return manager, group, target, fmt.Errorf("此解释器未安装 pip；请通过原管理器安装 pip 后刷新检查")
		}
		// The isolated metadata reader already bound pip to this interpreter's
		// site-packages. Starting `python -m pip` here would execute .pth/site
		// hooks before the user confirmed a plan. Inspect its entry as data.
		entry := filepath.Join(pip.Path, "pip", "__main__.py")
		if info, e := os.Stat(entry); e != nil || !info.Mode().IsRegular() {
			return manager, group, target, fmt.Errorf("此解释器的 pip 安装缺少入口文件，请通过原管理器修复")
		}
		manager = packageManager{Name: "pip", Executable: target.Interpreter, Version: pip.Version, Manifest: entry}
		target.Manager, target.ManagerPath = "pip", target.Interpreter
	}
	if group.Truncated {
		return manager, group, target, fmt.Errorf("所选环境超过 1024 个包，无法完整核验；请使用原包管理器")
	}
	return manager, group, target, nil
}

func inspectPackageTarget(ctx context.Context, target PackageTarget) PackageGroup {
	if target.Scope == "standalone_tool" {
		return inspectStandaloneUV(ctx, target)
	}
	if target.Ecosystem == "node" {
		if target.Scope == "project" {
			return inspectNodeProject(target.Root)
		}
		return inspectNodeGlobal(target.Root)
	}
	return inspectPythonPackages(ctx, target.Interpreter, target.Scope, target.Directory)
}

func resolveNodePackageManager(ctx context.Context, target PackageTarget) (packageManager, error) {
	manager := packageManager{Name: strings.Split(target.Manager, "@")[0]}
	if manager.Name != "npm" && manager.Name != "pnpm" {
		return manager, fmt.Errorf("此项目使用 %s，本版仅接入已安装的 npm/pnpm；请使用原管理器", target.Manager)
	}
	entry := target.ManagerPath
	if entry == "" {
		var e error
		entry, e = exec.LookPath(manager.Name)
		if e != nil {
			return manager, fmt.Errorf("未找到已安装的 %s；不会自动下载包管理器", manager.Name)
		}
	}
	if !filepath.IsAbs(entry) {
		return manager, fmt.Errorf("包管理器必须使用绝对路径")
	}
	manifests := nodeToolManifests(entry, manager.Name)
	if filepath.Base(entry) == "package.json" {
		manifests = append([]string{entry}, manifests...)
	}
	for _, path := range manifests {
		pkg, e := readNodeManifest(path)
		if e != nil || pkg.Name != manager.Name || pkg.Version == "" {
			continue
		}
		bin := "bin/npm-cli.js"
		if manager.Name == "pnpm" {
			bin = "bin/pnpm.cjs"
		}
		script := filepath.Join(filepath.Dir(path), filepath.FromSlash(bin))
		info, e := os.Stat(script)
		if e != nil || !info.Mode().IsRegular() {
			continue
		}
		manager.Manifest, manager.Script, manager.Version = path, script, pkg.Version
		break
	}
	if manager.Script == "" {
		return manager, fmt.Errorf("无法证明 %s 入口属于已安装的官方包；Corepack/未知启动器不会被自动执行，请安装原管理器后重查", manager.Name)
	}
	if protectedManagedPackagePath(manager.Manifest) && target.Tool != "" {
		return manager, fmt.Errorf("myEnv 固定后端或受管运行时中的管理器不能原地替换，请从工具链页切换运行时")
	}
	manager.Executable = target.Interpreter
	if manager.Executable == "" {
		var e error
		manager.Executable, e = exec.LookPath("node")
		if e != nil {
			return manager, fmt.Errorf("未找到用于运行 %s 的 Node.js", manager.Name)
		}
	}
	version, e := probeInventoryCommand(ctx, manager.Executable, []string{"--version"}, "", 8192)
	if e != nil || !inventoryVersionMatches("node", version) {
		return manager, fmt.Errorf("未能核验选定 Node.js")
	}
	probe := PackageCommand{Executable: manager.Executable, Args: []string{manager.Script, "--version"}, Directory: filepath.Dir(manager.Manifest)}
	output, e := readPackageCommand(ctx, target, manager, probe)
	if e != nil || strings.TrimSpace(output) != manager.Version {
		return manager, fmt.Errorf("%s 的实际版本与安装元数据不一致，请重新检查：%v", manager.Name, e)
	}
	return manager, nil
}

func packageToolTarget(ctx context.Context, target PackageTarget) (PackageTarget, error) {
	name := target.Tool
	if name == "npm" || name == "pnpm" {
		if target.ManagerPath == "" {
			return target, fmt.Errorf("请先刷新以确定管理工具的实际路径")
		}
		paths := nodeToolManifests(target.ManagerPath, name)
		if filepath.Base(target.ManagerPath) == "package.json" {
			paths = append([]string{target.ManagerPath}, paths...)
		}
		for _, manifest := range paths {
			pkg, e := readNodeManifest(manifest)
			if e != nil || pkg.Name != name {
				continue
			}
			root := filepath.Dir(filepath.Dir(manifest))
			if filepath.Base(root) != "node_modules" {
				continue
			}
			target.Root, target.Scope, target.Manager = root, "global_package_root", "npm"
			// npm owns npm-installed pnpm. Use its actual installed npm entry,
			// never execute a pnpm shim to provision or replace itself.
			if name == "pnpm" {
				target.ManagerPath = ""
			}
			return target, nil
		}
		return target, fmt.Errorf("未确认 %s 的安装所有者；请通过原安装器管理，不能直接删除入口文件", name)
	}
	if name == "pip" || name == "uv" {
		if target.Interpreter != "" {
			group := inspectPythonPackages(ctx, target.Interpreter, "interpreter", "")
			pkg := findManagedPackage(group, name)
			if pkg.Name != "" {
				target.Root, target.Scope, target.Manager = group.Root, "interpreter", "pip"
				return target, nil
			}
		}
		if name == "uv" {
			target.Scope = "standalone_tool"
			return target, nil
		}
		return target, fmt.Errorf("未确认 pip 所属解释器，请刷新本机检查")
	}
	return target, fmt.Errorf("此管理工具尚无可核验的原管理器适配")
}

func protectedManagedPackagePath(path string) bool {
	if resolved, e := config.ResolveExistingPath(path); e == nil {
		path = resolved
	}
	path = "/" + strings.Trim(strings.ToLower(filepath.ToSlash(path)), "/") + "/"
	return strings.Contains(path, "/.myenv/") || strings.Contains(path, "/.myenv-profile/") || strings.Contains(path, "/myenv/runtimes/") || strings.Contains(path, "/myenv/backends/")
}

func nodeGlobalPrefix(root string) (string, error) {
	prefix := filepath.Dir(root)
	if runtime.GOOS != "windows" {
		if filepath.Base(prefix) != "lib" {
			return "", fmt.Errorf("此目录不是 npm 的 lib/node_modules 全局位置")
		}
		prefix = filepath.Dir(prefix)
	}
	if filepath.Dir(prefix) == prefix {
		return "", fmt.Errorf("磁盘根目录不能作为 npm 全局前缀")
	}
	return prefix, nil
}

func checkPackageWorkspace(root string) error {
	for _, name := range []string{"package.json", "package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", ".npmrc", ".pnpmfile.cjs"} {
		info, e := os.Lstat(filepath.Join(root, name))
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s 是链接或特殊文件，请通过项目终端管理，避免写入其他目录", name)
		}
	}
	if modules, e := config.ResolveExistingPath(filepath.Join(root, "node_modules")); e == nil && !packagePathWithin(root, modules) {
		return fmt.Errorf("项目 node_modules 指向其他目录，请通过原管理器管理")
	}
	data, err := readInventoryFile(filepath.Join(root, "package.json"))
	if err != nil {
		return err
	}
	var manifest map[string]json.RawMessage
	if err = json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if value := manifest["workspaces"]; len(value) > 0 && string(value) != "null" {
		return fmt.Errorf("此项目是多工作区仓库；本版请在项目终端使用原管理器，避免批量修改其他工作区")
	}
	for current, depth := root, 0; depth < 12; depth++ {
		if _, e := os.Stat(filepath.Join(current, "pnpm-workspace.yaml")); e == nil {
			return fmt.Errorf("此项目属于 pnpm 工作区；请在项目终端通过原管理器选择工作区")
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return nil
}

func (manager packageManager) command(target PackageTarget, operation string, change PackageChange) (PackageCommand, error) {
	command := PackageCommand{Executable: manager.Executable, Directory: target.Root, Package: change.Name}
	if target.Scope == "standalone_tool" {
		if operation == "remove" {
			return command, fmt.Errorf("uv 独立安装器没有卸载命令；请通过其原安装方式移除，myEnv 不直接删除可执行文件")
		}
		command.Args = []string{"self", "update", change.Version, "--no-config", "--no-progress"}
		command.Directory = filepath.Dir(manager.Executable)
		return command, nil
	}
	if target.Ecosystem == "python" {
		command.Args = []string{"-I", "-m", "pip", "--isolated", "--disable-pip-version-check", "--no-input"}
		if operation == "remove" {
			command.Args = append(command.Args, "uninstall", "--yes", change.Name)
		} else {
			command.Args = append(command.Args, "install", "--upgrade", "--only-binary=:all:", "--index-url", "https://pypi.org/simple", change.Name+"=="+change.Version)
		}
		return command, nil
	}
	command.Args = []string{manager.Script}
	if manager.Name == "npm" {
		action := "install"
		if operation == "remove" {
			action = "uninstall"
		}
		command.Args = append(command.Args, action, "--ignore-scripts", "--no-audit", "--no-fund", "--registry=https://registry.npmjs.org")
		if target.Scope == "global_package_root" {
			prefix, e := nodeGlobalPrefix(target.Root)
			if e != nil {
				return command, e
			}
			command.Directory = prefix
			command.Args = append(command.Args, "--global", "--prefix", prefix)
		} else {
			command.Directory = target.Directory
			command.Args = append(command.Args, "--prefix", target.Directory, "--workspaces=false")
			if operation != "remove" {
				command.Args = append(command.Args, "--save-exact")
			}
		}
	} else {
		action := "add"
		if operation == "remove" {
			action = "remove"
		}
		command.Directory = target.Directory
		command.Args = append(command.Args, action)
		if operation != "remove" {
			command.Args = append(command.Args, "--ignore-scripts", "--ignore-pnpmfile", "--registry=https://registry.npmjs.org", "--save-exact")
		}
	}
	if strings.HasPrefix(change.Name, "@") && (manager.Name == "npm" || operation != "remove") {
		scope := strings.Split(change.Name, "/")[0]
		command.Args = append(command.Args, "--"+scope+":registry=https://registry.npmjs.org")
	}
	if target.Scope == "project" && operation != "remove" {
		switch change.Kind {
		case "dependency":
			command.Args = append(command.Args, "--save-prod")
		case "development":
			command.Args = append(command.Args, "--save-dev")
		case "optional":
			command.Args = append(command.Args, "--save-optional")
		case "global", "":
		default:
			return command, fmt.Errorf("未知依赖类型：%s", change.Kind)
		}
	}
	if operation == "remove" {
		command.Args = append(command.Args, change.Name)
	} else {
		command.Args = append(command.Args, change.Name+"@"+change.Version)
	}
	return command, nil
}

func packageEnvironment(target PackageTarget, manager packageManager) ([]string, error) {
	parent := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "NPM_CONFIG_") || strings.HasPrefix(upper, "PNPM_") || strings.HasPrefix(upper, "PIP_") || strings.HasPrefix(upper, "PYTHON") || strings.HasPrefix(upper, "UV_") || strings.HasPrefix(upper, "AXOUPDATER_") || upper == "NODE_OPTIONS" {
			continue
		}
		parent = append(parent, entry)
	}
	overlay := map[string]string{"CI": "1", "NPM_CONFIG_USERCONFIG": os.DevNull, "NPM_CONFIG_UPDATE_NOTIFIER": "false", "NPM_CONFIG_ENGINE_STRICT": "true", "NPM_CONFIG_IGNORE_SCRIPTS": "true", "NPM_CONFIG_IGNORE_PNPMFILE": "true", "COREPACK_ENABLE_NETWORK": "0", "PIP_CONFIG_FILE": os.DevNull, "PIP_DISABLE_PIP_VERSION_CHECK": "1", "UV_NO_MODIFY_PATH": "1", "UV_NO_PROGRESS": "1"}
	if manager.Name != "pnpm" {
		delete(overlay, "NPM_CONFIG_IGNORE_PNPMFILE")
	}
	return runner.Environment(parent, overlay, []string{filepath.Dir(manager.Executable)}, runtime.GOOS == "windows")
}
func readPackageCommand(ctx context.Context, target PackageTarget, manager packageManager, command PackageCommand) (string, error) {
	env, err := packageEnvironment(target, manager)
	if err != nil {
		return "", err
	}
	out := &inventoryOutput{limit: 8192}
	diagnostic := &inventoryOutput{limit: 8192}
	probe, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	code, err := runner.Execute(probe, runner.Process{Background: true, Executable: command.Executable, Args: command.Args, Directory: command.Directory, Environment: env, Stdout: out, Stderr: diagnostic})
	if probe.Err() != nil {
		return "", probe.Err()
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err == nil && code != 0 {
		err = fmt.Errorf("管理器核验退出码 %d：%s", code, diagnostic.text.String())
	}
	if out.truncated {
		err = fmt.Errorf("管理器核验输出超限")
	}
	return strings.TrimSpace(out.text.String()), err
}
