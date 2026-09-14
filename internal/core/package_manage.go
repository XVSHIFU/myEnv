package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"myenv/internal/backend"
	"myenv/internal/runner"
)

// PackageTarget names one concrete installation. Root is the package root or
// interpreter prefix; Directory is separately the project's manifest directory.
type PackageTarget struct {
	Ecosystem   string `json:"ecosystem"`
	Scope       string `json:"scope"`
	Root        string `json:"root"`
	Directory   string `json:"directory,omitempty"`
	Interpreter string `json:"interpreter,omitempty"`
	Manager     string `json:"manager,omitempty"`
	ManagerPath string `json:"manager_path,omitempty"`
	Tool        string `json:"tool,omitempty"`
}
type PackageItemRequest struct {
	Name    string `json:"name"`
	Desired string `json:"desired,omitempty"`
	Kind    string `json:"kind,omitempty"`
}
type PackageRequest struct {
	Target    PackageTarget        `json:"target"`
	Operation string               `json:"operation"`
	Items     []PackageItemRequest `json:"items"`
}
type PackageChange struct {
	Name        string `json:"name"`
	Before      string `json:"before,omitempty"`
	Requested   string `json:"requested,omitempty"`
	Desired     string `json:"desired,omitempty"`
	Version     string `json:"version,omitempty"`
	Kind        string `json:"kind,omitempty"`
	OfficialURL string `json:"official_url,omitempty"`
}
type PackageCommand struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
	Directory  string   `json:"directory"`
	Package    string   `json:"package"`
}
type PackagePlan struct {
	ID          string           `json:"id"`
	Request     PackageRequest   `json:"request"`
	Changes     []PackageChange  `json:"changes"`
	Commands    []PackageCommand `json:"commands"`
	Warnings    []string         `json:"warnings"`
	StateDigest string           `json:"state_digest"`
	CreatedAt   time.Time        `json:"created_at"`
	manager     packageManager
}
type PackageOutcome struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Version string `json:"version,omitempty"`
	Problem string `json:"problem,omitempty"`
}
type PackageApplyResult struct {
	PlanID       string           `json:"plan_id"`
	Outcomes     []PackageOutcome `json:"outcomes"`
	Log          string           `json:"log,omitempty"`
	LogTruncated bool             `json:"log_truncated,omitempty"`
	Group        PackageGroup     `json:"group"`
}

type packageProgressKey struct{}
type packageLogKey struct{}

func WithPackageProgress(ctx context.Context, callback func(string)) context.Context {
	return context.WithValue(ctx, packageProgressKey{}, callback)
}
func WithPackageLog(ctx context.Context, writer io.Writer) context.Context {
	return context.WithValue(ctx, packageLogKey{}, writer)
}
func packageProgress(ctx context.Context, label string) {
	if callback, ok := ctx.Value(packageProgressKey{}).(func(string)); ok && callback != nil {
		callback(label)
	}
}

// Plans are retained by the caller in memory. Apply never accepts a replacement
// request or resolves a moving latest tag a second time.
func (s *Service) PlanPackages(ctx context.Context, request PackageRequest) (PackagePlan, error) {
	registry := backend.PackageRegistry{Client: &http.Client{Timeout: 45 * time.Second}}
	if s.Node != nil && s.Node.Client != nil {
		registry.Client = s.Node.Client
	}
	return s.planPackages(ctx, request, registry)
}

type packageRegistry interface {
	Package(context.Context, string, string) (backend.RegistryPackage, error)
	Version(context.Context, string, string, string) (backend.RegistryPackageVersion, error)
}

func (s *Service) planPackages(ctx context.Context, request PackageRequest, registry packageRegistry) (PackagePlan, error) {
	request.Items = append([]PackageItemRequest(nil), request.Items...)
	plan := PackagePlan{Request: request, Changes: []PackageChange{}, Commands: []PackageCommand{}, Warnings: []string{}, CreatedAt: time.Now().UTC()}
	if request.Operation != "install" && request.Operation != "remove" && request.Operation != "update" && request.Operation != "switch" {
		return plan, fmt.Errorf("请选择安装、更新、切换版本或卸载")
	}
	if len(request.Items) == 0 || len(request.Items) > 100 {
		return plan, fmt.Errorf("每次请选择 1–100 个包")
	}
	seen := map[string]bool{}
	for _, item := range request.Items {
		name := item.Name
		if request.Target.Ecosystem == "python" {
			name = normalizePythonPackage(name)
		}
		if !validManagePackage(request.Target.Ecosystem, name) || seen[name] {
			return plan, fmt.Errorf("包名无效或重复：%s", item.Name)
		}
		seen[name] = true
	}
	manager, group, target, err := s.packageTarget(ctx, request.Target)
	if err != nil {
		return plan, err
	}
	plan.Request.Target, plan.manager = target, manager
	plan.Warnings = append(plan.Warnings, "由选定包管理器直接修改此环境，不属于 myEnv 环境代回退；失败或取消后将重新检查每个包。")
	if target.Scope == "standalone_tool" {
		plan.Warnings = append(plan.Warnings, "由官方 uv 独立安装器更新已核验的安装位置；不修改 PATH 或 shell 配置。")
	} else if target.Ecosystem == "node" {
		plan.Warnings = append(plan.Warnings, "安装脚本已禁用。需要编译或安装脚本的包可能需要在明确的项目终端中完成构建。")
	} else {
		plan.Warnings = append(plan.Warnings, "仅安装官方 PyPI 的二进制 wheel，不运行源码构建；不修改终端默认 Python。")
	}
	if target.Ecosystem == "python" && target.Scope == "project" {
		plan.Warnings = append(plan.Warnings, "此操作修改所选外部虚拟环境中的包；不会自动改写 pyproject.toml 或 requirements.txt。")
	}
	if target.Scope != "project" {
		plan.Warnings = append(plan.Warnings, "“最新版”在本次预览时解析为具体版本；不会在后台自动更新。")
	}
	for index, item := range request.Items {
		packageProgress(ctx, fmt.Sprintf("核验 %s · %d/%d", item.Name, index+1, len(request.Items)))
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		if target.Ecosystem == "python" {
			item.Name = normalizePythonPackage(item.Name)
		}
		current := findManagedPackage(group, item.Name)
		if len(request.Items) > 1 && (item.Name == manager.Name || target.Scope == "standalone_tool") {
			return plan, fmt.Errorf("%s 是本次操作所用的管理器，请单独管理，避免中途替换执行工具", item.Name)
		}
		if target.Tool != "" && (len(request.Items) != 1 || item.Name != target.Tool) {
			return plan, fmt.Errorf("管理工具入口只允许操作已选中的工具")
		}
		if current.Path != "" && target.Ecosystem == "python" && !packagePathWithin(target.Root, current.Path) {
			return plan, fmt.Errorf("%s 来自此环境继承的其他解释器；请在其所属环境中管理", item.Name)
		}
		change := PackageChange{Name: item.Name, Before: current.Version, Requested: current.Requested, Desired: item.Desired, Kind: item.Kind}
		if change.Kind == "" {
			change.Kind = current.Kind
		}
		if change.Kind == "" {
			change.Kind = "dependency"
		}
		if request.Operation == "remove" {
			if current.Name == "" {
				return plan, fmt.Errorf("%s 已不在所选环境中，请刷新列表", item.Name)
			}
		} else {
			if item.Desired == "" {
				item.Desired = "latest"
			}
			change.Desired = item.Desired
			version := item.Desired
			if version == "latest" {
				metadata, e := registry.Package(ctx, target.Ecosystem, item.Name)
				if e != nil {
					return plan, e
				}
				if metadata.State == "not_found" {
					return plan, fmt.Errorf("OFFICIAL_NOT_FOUND: 官方未找到 %s；仍可卸载本地安装", item.Name)
				}
				version = metadata.Latest
				if version == "" {
					return plan, fmt.Errorf("%s 官方未提供可用稳定版本，请选择具体版本", item.Name)
				}
			}
			if !validManageVersion(version) {
				return plan, fmt.Errorf("请选择具体版本或 latest；不接受 URL、文件或命令：%s", version)
			}
			release, e := registry.Version(ctx, target.Ecosystem, item.Name, version)
			if e != nil {
				return plan, e
			}
			if release.State == "not_found" || !release.Available || release.Yanked {
				return plan, fmt.Errorf("OFFICIAL_NOT_FOUND: 官方未找到可安装的 %s %s", item.Name, version)
			}
			change.Version, change.OfficialURL = release.Version, release.URL
			if release.Deprecated != "" {
				plan.Warnings = append(plan.Warnings, item.Name+" 已被上游标记弃用："+release.Deprecated)
			}
		}
		plan.Request.Items[index] = item
		plan.Changes = append(plan.Changes, change)
		command, err := manager.command(target, request.Operation, change)
		if err != nil {
			return plan, err
		}
		plan.Commands = append(plan.Commands, command)
	}
	plan.StateDigest, err = packageStateDigest(target, manager, group)
	if err != nil {
		return plan, err
	}
	plan.ID = packagePlanDigest(plan)
	return plan, nil
}

func packagePathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

var packageApplyMu sync.Mutex

func (s *Service) ApplyPackages(ctx context.Context, plan PackagePlan) (PackageApplyResult, error) {
	result := PackageApplyResult{PlanID: plan.ID, Outcomes: []PackageOutcome{}}
	if plan.ID == "" || packagePlanDigest(plan) != plan.ID || len(plan.Commands) != len(plan.Changes) {
		return result, fmt.Errorf("计划无效，请重新预览")
	}
	// GUI operations are serial already; protect direct core callers too. Waiting
	// must remain cancellable without holding an environment mutation indefinitely.
	if !packageApplyMu.TryLock() {
		return result, fmt.Errorf("已有包管理操作正在执行，请完成或取消后重试")
	}
	defer packageApplyMu.Unlock()
	manager, group, target, err := s.packageTarget(ctx, plan.Request.Target)
	if err != nil {
		return result, err
	}
	digest, err := packageStateDigest(target, manager, group)
	if err != nil {
		return result, err
	}
	if digest != plan.StateDigest {
		return result, fmt.Errorf("INPUT_CHANGED: 环境、管理器或依赖声明已改变，请刷新并重新预览")
	}
	output := &inventoryOutput{limit: 256 * 1024}
	var logWriter io.Writer = output
	if writer, ok := ctx.Value(packageLogKey{}).(io.Writer); ok && writer != nil {
		logWriter = io.MultiWriter(output, writer)
	}
	defer func() { result.Log = output.text.String(); result.LogTruncated = output.truncated }()
	var applyErr error
	for i, command := range plan.Commands {
		packageProgress(ctx, fmt.Sprintf("处理 %s · %d/%d", plan.Changes[i].Name, i+1, len(plan.Commands)))
		if ctx.Err() != nil {
			applyErr = ctx.Err()
			break
		}
		expected, e := manager.command(target, plan.Request.Operation, plan.Changes[i])
		if e != nil || !packageCommandsEqual(expected, command) {
			return result, fmt.Errorf("计划命令与选定环境不匹配，请重新预览")
		}
		env, e := packageEnvironment(target, manager)
		if e != nil {
			return result, e
		}
		_, _ = fmt.Fprintf(logWriter, "\n%s\n", plan.Changes[i].Name)
		code, e := runner.Execute(ctx, runner.Process{Background: true, Executable: command.Executable, Args: command.Args, Directory: command.Directory, Environment: env, Stdout: logWriter, Stderr: logWriter})
		if ctx.Err() != nil {
			e = ctx.Err()
		}
		if e == nil && code != 0 {
			e = fmt.Errorf("包管理器退出码 %d", code)
		}
		if e != nil {
			applyErr = e
			break
		}
		if target.Ecosystem == "node" && target.Scope == "project" && plan.Request.Operation != "remove" && plan.Changes[i].Desired == "latest" {
			if e = setNodeLatestExpectation(target, plan.Changes[i]); e != nil {
				applyErr = e
				break
			}
		}
	}
	// Cancellation does not hide a manager's partial changes. Inspection gets a
	// separate short deadline after the supervised process tree has stopped.
	check, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	packageProgress(ctx, "重新检查实际安装结果")
	defer cancel()
	result.Group = inspectPackageTarget(check, target)
	for _, change := range plan.Changes {
		current := findManagedPackage(result.Group, change.Name)
		outcome := PackageOutcome{Name: change.Name, Version: current.Version, State: "not_applied"}
		if result.Group.State != "available" {
			outcome.State = "unverified"
			outcome.Problem = result.Group.Problem
		} else if plan.Request.Operation == "remove" && current.Name == "" {
			outcome.State = "removed"
		} else if plan.Request.Operation != "remove" && current.Version == change.Version {
			outcome.State = "applied"
			if target.Ecosystem == "node" && target.Scope == "project" {
				expected := change.Version
				if change.Desired == "latest" {
					expected = "latest"
				}
				if current.Requested != expected || current.Kind != change.Kind {
					outcome.State = "changed"
					outcome.Problem = "已安装版本，但依赖类型或期望未达到预览状态，请刷新检查"
				}
			}
		} else if current.Version != change.Before {
			outcome.State = "changed"
		}
		if applyErr != nil && outcome.State != "applied" && outcome.State != "removed" {
			outcome.Problem = applyErr.Error()
		}
		result.Outcomes = append(result.Outcomes, outcome)
	}
	result.Log, result.LogTruncated = output.text.String(), output.truncated
	if applyErr == nil {
		for _, outcome := range result.Outcomes {
			if outcome.State != "applied" && outcome.State != "removed" {
				applyErr = fmt.Errorf("操作结束，但部分包未达到预览版本，请查看结果并刷新")
				break
			}
		}
	}
	return result, applyErr
}

func validManagePackage(ecosystem, name string) bool {
	if len(name) > 214 {
		return false
	}
	if ecosystem == "node" {
		return regexp.MustCompile(`^(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`).MatchString(name)
	}
	return ecosystem == "python" && regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`).MatchString(name)
}
func validManageVersion(value string) bool {
	return len(value) <= 100 && regexp.MustCompile(`^[0-9][A-Za-z0-9.!+_-]*$`).MatchString(value)
}
func findManagedPackage(group PackageGroup, name string) InventoryPackage {
	for _, p := range group.Packages {
		if p.Name == name || group.Ecosystem == "python" && normalizePythonPackage(p.Name) == normalizePythonPackage(name) {
			return p
		}
	}
	return InventoryPackage{}
}
func packagePlanDigest(plan PackagePlan) string {
	plan.ID = ""
	data, _ := json.Marshal(plan)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func packageCommandsEqual(a, b PackageCommand) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func packageStateDigest(target PackageTarget, manager packageManager, group PackageGroup) (string, error) {
	hash := sha256.New()
	base, _ := json.Marshal(struct {
		Target  PackageTarget
		Manager packageManager
		Group   PackageGroup
	}{target, manager, group})
	_, _ = hash.Write(base)
	paths := []string{manager.Executable, manager.Script, manager.Manifest, manager.Receipt, target.Interpreter}
	if target.Scope == "project" {
		for _, name := range []string{"package.json", "package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc", ".pnpmfile.cjs", "pyproject.toml", "uv.lock", "requirements.txt"} {
			paths = append(paths, filepath.Join(target.Directory, name))
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		if path == "" {
			continue
		}
		_, _ = io.WriteString(hash, path)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			_, _ = io.WriteString(hash, "missing")
			continue
		}
		if err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(hash, "%d:%d:%s", info.Size(), info.ModTime().UnixNano(), info.Mode())
		if path == manager.Executable || path == target.Interpreter {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		n, e := io.Copy(hash, io.LimitReader(f, 16*1024*1024+1))
		_ = f.Close()
		if e != nil {
			return "", e
		}
		if n > 16*1024*1024 {
			return "", fmt.Errorf("依赖元数据超过 16 MiB，请在项目终端中管理")
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
