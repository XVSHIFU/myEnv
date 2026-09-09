package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type CompilerCheck struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Path   string `json:"path,omitempty"`
	Advice string `json:"advice"`
}

// Presence is not a compile test. Never invokes cl/link or installs a workload.
func CompilerPrerequisites(ctx context.Context) []CompilerCheck {
	checks := []CompilerCheck{}
	for _, name := range []string{"clang", "gcc", "g++", "cmake"} {
		row := CompilerCheck{Name: name, State: "not_found", Advice: "未在 PATH 中找到；这不是项目必须安装全部工具的意思。"}
		if p, e := exec.LookPath(name); e == nil && filepath.IsAbs(p) {
			row.Path = p
			row.State = "present"
			row.Advice = "已找到入口；尚未验证实际编译。"
		}
		checks = append(checks, row)
	}
	if runtime.GOOS != "windows" {
		return checks
	}
	msvc := CompilerCheck{Name: "MSVC", State: "not_found", Advice: "Windows Rust 原生依赖可能需要 Build Tools 的 Desktop development with C++ 工作负载：https://visualstudio.microsoft.com/visual-cpp-build-tools/"}
	vswhere := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft Visual Studio", "Installer", "vswhere.exe")
	if i, e := os.Stat(vswhere); e == nil && i.Mode().IsRegular() {
		if b, e := managerRead(ctx, vswhere, filepath.Dir(vswhere), "-latest", "-products", "*", "-requires", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64", "-property", "installationPath"); e == nil {
			p := strings.TrimSpace(string(b))
			if filepath.IsAbs(p) {
				msvc.Path = p
				msvc.State = "registered"
				msvc.Advice = "Visual Studio Installer 已登记 C++ 工具组件；不等同于所有链接场景已验证。"
			}
		}
	}
	checks = append(checks, msvc)
	sdk := CompilerCheck{Name: "Windows SDK", State: "not_found", Advice: "通过微软 Build Tools 安装器选择 Windows SDK；myenv 不会自动安装。https://developer.microsoft.com/windows/downloads/windows-sdk/"}
	root := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Windows Kits", "10", "Lib")
	if dir, e := os.Open(root); e == nil {
		entries, e := dir.ReadDir(256)
		dir.Close()
		if e == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				base := filepath.Join(root, entry.Name())
				a, ea := os.Stat(filepath.Join(base, "um", "x64", "kernel32.lib"))
				b, eb := os.Stat(filepath.Join(base, "ucrt", "x64", "ucrt.lib"))
				if ea == nil && eb == nil && a.Mode().IsRegular() && b.Mode().IsRegular() {
					sdk.Path = base
					sdk.State = "present"
					sdk.Advice = "找到 x64 Windows/UCRT 链接库；未做完整 SDK 验证。"
					break
				}
			}
		}
	}
	return append(checks, sdk)
}
