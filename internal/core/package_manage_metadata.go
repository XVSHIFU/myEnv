package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Latest is a declaration policy, while the native lock keeps the exact version
// that the user reviewed. Never run a second install against a moving tag.
func setNodeLatestExpectation(target PackageTarget, change PackageChange) error {
	section := "dependencies"
	if change.Kind == "development" {
		section = "devDependencies"
	}
	if change.Kind == "optional" {
		section = "optionalDependencies"
	}
	manifestPath := filepath.Join(target.Directory, "package.json")
	manifestData, err := readInventoryFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest map[string]any
	if err = json.Unmarshal(manifestData, &manifest); err != nil {
		return err
	}
	deps, ok := manifest[section].(map[string]any)
	if !ok || deps[change.Name] != change.Version {
		return fmt.Errorf("INPUT_CHANGED: 包已安装，但依赖声明与预览不一致；未改写 latest")
	}
	deps[change.Name] = "latest"
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if target.Manager == "npm" {
		// npm gives shrinkwrap precedence and may leave package-lock untouched.
		// Only rewrite the lock that actually governed this manager invocation.
		lockName := "package-lock.json"
		if _, e := os.Stat(filepath.Join(target.Directory, "npm-shrinkwrap.json")); e == nil {
			lockName = "npm-shrinkwrap.json"
		}
		for _, name := range []string{lockName} {
			path := filepath.Join(target.Directory, name)
			data, e := readInventoryFile(path)
			if os.IsNotExist(e) {
				continue
			}
			if e != nil {
				return e
			}
			var lock map[string]any
			if e = json.Unmarshal(data, &lock); e != nil {
				return e
			}
			packages, _ := lock["packages"].(map[string]any)
			root, _ := packages[""].(map[string]any)
			values, _ := root[section].(map[string]any)
			if values == nil || values[change.Name] != change.Version {
				return fmt.Errorf("包已安装，但原生锁格式不支持保留 latest 期望；请刷新检查")
			}
			values[change.Name] = "latest"
			next, e := json.MarshalIndent(lock, "", "  ")
			if e != nil {
				return e
			}
			current, e := readInventoryFile(manifestPath)
			if e != nil {
				return e
			}
			if !bytes.Equal(current, manifestData) {
				return fmt.Errorf("INPUT_CHANGED: 依赖声明在保存 latest 前已改变")
			}
			if e = replacePackageMetadata(path, data, append(next, '\n')); e != nil {
				return e
			}
		}
	} else {
		path := filepath.Join(target.Directory, "pnpm-lock.yaml")
		data, e := readInventoryFile(path)
		if e != nil {
			return e
		}
		var document yaml.Node
		if e = yaml.Unmarshal(data, &document); e != nil {
			return e
		}
		if len(document.Content) == 0 {
			return fmt.Errorf("pnpm 锁为空")
		}
		root := document.Content[0]
		importers := packageYAMLField(root, "importers")
		entry := packageYAMLField(importers, ".")
		entry = packageYAMLField(entry, section)
		entry = packageYAMLField(entry, change.Name)
		spec := packageYAMLField(entry, "specifier")
		if spec == nil || spec.Value != change.Version {
			return fmt.Errorf("包已安装，但 pnpm 锁格式不支持保留 latest 期望；请刷新检查")
		}
		spec.Value = "latest"
		spec.Tag = "!!str"
		next, e := yaml.Marshal(&document)
		if e != nil {
			return e
		}
		current, e := readInventoryFile(manifestPath)
		if e != nil {
			return e
		}
		if !bytes.Equal(current, manifestData) {
			return fmt.Errorf("INPUT_CHANGED: 依赖声明在保存 latest 前已改变")
		}
		if e = replacePackageMetadata(path, data, next); e != nil {
			return e
		}
	}
	return replacePackageMetadata(manifestPath, manifestData, encoded)
}
func packageYAMLField(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
func replacePackageMetadata(path string, before, after []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("依赖元数据不是普通文件")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".myenv-package-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(info.Mode().Perm()); err == nil {
		_, err = f.Write(after)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	latest, err := readInventoryFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(latest, before) {
		return fmt.Errorf("INPUT_CHANGED: 依赖文件已被其他程序修改")
	}
	return os.Rename(f.Name(), path)
}

type standaloneUVReceipt struct {
	InstallPrefix string   `json:"install_prefix"`
	InstallLayout string   `json:"install_layout"`
	Binaries      []string `json:"binaries"`
	Source        struct {
		AppName     string `json:"app_name"`
		Name        string `json:"name"`
		Owner       string `json:"owner"`
		ReleaseType string `json:"release_type"`
	} `json:"source"`
}

func standaloneUVTarget(ctx context.Context, target PackageTarget) (packageManager, PackageGroup, PackageTarget, error) {
	manager := packageManager{Name: "uv", Executable: target.ManagerPath}
	group := PackageGroup{}
	if target.Tool != "uv" || !filepath.IsAbs(manager.Executable) || protectedManagedPackagePath(manager.Executable) {
		return manager, group, target, fmt.Errorf("不能修改 myEnv 固定 uv 后端或未知的 uv 入口")
	}
	info, err := os.Lstat(manager.Executable)
	if err != nil || !info.Mode().IsRegular() {
		return manager, group, target, fmt.Errorf("uv 必须是已核验的独立可执行文件")
	}
	paths := []string{}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(xdg) {
		paths = append(paths, filepath.Join(xdg, "uv", "uv-receipt.json"))
	}
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); filepath.IsAbs(base) {
			paths = append(paths, filepath.Join(base, "uv", "uv-receipt.json"))
		}
	} else {
		if home, e := os.UserHomeDir(); e == nil {
			paths = append(paths, filepath.Join(home, ".config", "uv", "uv-receipt.json"))
		}
	}
	for _, path := range paths {
		data, e := readInventoryFile(path)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return manager, group, target, e
		}
		var receipt standaloneUVReceipt
		if e = json.Unmarshal(data, &receipt); e != nil {
			return manager, group, target, e
		}
		if receipt.Source.AppName != "uv" || receipt.Source.Name != "uv" || receipt.Source.Owner != "astral-sh" || receipt.Source.ReleaseType != "github" || receipt.InstallLayout != "flat" || !sameInventoryPath(receipt.InstallPrefix, filepath.Dir(manager.Executable)) {
			return manager, group, target, fmt.Errorf("uv 安装收据不属于此入口或官方来源；请用原安装方式管理")
		}
		found := false
		for _, name := range receipt.Binaries {
			if name == filepath.Base(manager.Executable) {
				found = true
			}
		}
		if !found {
			return manager, group, target, fmt.Errorf("uv 安装收据未列出当前入口")
		}
		manager.Receipt = path
		break
	}
	if manager.Receipt == "" {
		return manager, group, target, fmt.Errorf("未找到此 uv 的独立安装收据；若由 pip 安装，请选择它所属 Python 解释器中的 uv 包管理")
	}
	target.Root, target.Manager, target.Scope = filepath.Dir(manager.Executable), "uv", "standalone_tool"
	group = inspectStandaloneUV(ctx, target)
	if group.State != "available" {
		return manager, group, target, fmt.Errorf("不能核验 uv：%s", group.Problem)
	}
	manager.Version = group.Packages[0].Version
	return manager, group, target, nil
}
func inspectStandaloneUV(ctx context.Context, target PackageTarget) PackageGroup {
	group := PackageGroup{Ecosystem: "python", Scope: "standalone_tool", Root: target.Root, Manager: "uv", State: "unverified", Packages: []InventoryPackage{}}
	version, err := probeInventoryCommand(ctx, target.ManagerPath, []string{"--version"}, filepath.Dir(target.ManagerPath), 8192)
	if err != nil {
		group.Problem = err.Error()
		return group
	}
	parts := strings.Fields(version)
	if len(parts) < 2 || parts[0] != "uv" {
		group.Problem = "输出未确认 uv 身份"
		return group
	}
	group.State = "available"
	group.Packages = append(group.Packages, InventoryPackage{Name: "uv", Version: parts[1], Path: target.ManagerPath, Kind: "tool", State: "installed"})
	return group
}
