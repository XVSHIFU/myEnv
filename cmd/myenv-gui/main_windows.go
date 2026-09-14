//go:build windows && gui

package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	win "github.com/wailsapp/wails/v2/pkg/options/windows"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"io/fs"
	"myenv/internal/cli"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

//go:embed all:frontend/dist
var assets embed.FS
var version = "0.1.0-dev"

type App struct {
	controller *cli.UIController
	ctx        context.Context
	closing    atomic.Bool
	ready      atomic.Bool
	stopped    chan struct{}
}

func (a *App) Start(r cli.UIRequest) (uint64, error) { return a.controller.Start(r) }
func (a *App) Current() cli.UITask                   { return a.controller.Current() }
func (a *App) Cancel(id uint64)                      { a.controller.Cancel(id) }
func (a *App) Confirm(id uint64, allow bool)         { a.controller.Confirm(id, allow) }
func (a *App) OpenProject() (string, error) {
	return wr.OpenDirectoryDialog(a.ctx, wr.OpenDialogOptions{Title: "打开 myEnv 项目"})
}
func (a *App) OpenManager() (string, error) {
	return wr.OpenFileDialog(a.ctx, wr.OpenDialogOptions{Title: "选择原管理器可执行文件"})
}
func (a *App) Version() string { return version }
func (a *App) NormalizeDirectory(path string) (cli.UIDirectory, error) {
	return cli.NormalizeUIDirectory(path)
}
func (a *App) History() []cli.UITask      { return a.controller.History() }
func (a *App) CopyPath(path string) error { return wr.ClipboardSetText(a.ctx, path) }
func (a *App) RevealPath(path string) error {
	directory, err := cli.NormalizeUIDirectory(path)
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("open")
	target, err := windows.UTF16PtrFromString(directory.Path)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL)
}
func (a *App) Run(directory string, global bool, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("请指定运行命令")
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe = filepath.Join(filepath.Dir(exe), "myenv.exe")
	if _, err = os.Stat(exe); err != nil {
		return "", fmt.Errorf("缺少随包同版本 myenv.exe: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	check := exec.CommandContext(checkCtx, exe, "--version")
	check.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := check.Output()
	if err != nil || string(out) != "myenv version "+version+"\n" {
		return "", fmt.Errorf("随包 CLI 版本不匹配")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	argv := []string{"-C", absolute, "run", "--hold-console"}
	if global {
		argv = append(argv, "--global")
	}
	argv = append(argv, "--")
	argv = append(argv, args...)
	// No STARTF_USESTDHANDLES: the new console supplies real interactive handles.
	// exec.Cmd's nil streams become NUL and would break input in this console.
	parts := []string{syscall.EscapeArg(exe)}
	for _, arg := range argv {
		parts = append(parts, syscall.EscapeArg(arg))
	}
	application, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return "", err
	}
	line, err := windows.UTF16PtrFromString(strings.Join(parts, " "))
	if err != nil {
		return "", err
	}
	cwd, err := windows.UTF16PtrFromString(absolute)
	if err != nil {
		return "", err
	}
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var process windows.ProcessInformation
	if err = windows.CreateProcess(application, line, nil, nil, false, windows.CREATE_NEW_CONSOLE, nil, cwd, &startup, &process); err != nil {
		return "", err
	}
	windows.CloseHandle(process.Thread)
	windows.CloseHandle(process.Process)
	return "已交给终端；结果保留到按 Enter 关闭，关闭 GUI 不会取消它。", nil
}

var onGUIReady = func(context.Context) {}
var guiNamespace string
var guiWebviewPath string
var onGUIShutdown = func(cli.UITask) {}

func main() {
	a := &App{controller: cli.NewUIController(guiNamespace), stopped: make(chan struct{})}
	content, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		panic(err)
	}
	data, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	webviewPath := filepath.Join(data, "myenv", "webview")
	if guiWebviewPath != "" {
		webviewPath = guiWebviewPath
	}
	err = wails.Run(&options.App{Title: "myEnv", Width: 1100, Height: 780, MinWidth: 760, MinHeight: 540, AssetServer: &assetserver.Options{Assets: content}, Bind: []interface{}{a},
		Windows:    &win.Options{WebviewUserDataPath: webviewPath, IsZoomControlEnabled: true, Theme: win.SystemDefault, CustomTheme: nativeWindowTheme()},
		OnDomReady: onGUIReady,
		OnStartup: func(ctx context.Context) {
			a.ctx = ctx
			go func() {
				for {
					select {
					case <-a.stopped:
						return
					case <-a.controller.Updates():
						wr.EventsEmit(ctx, "task", a.Current())
					}
				}
			}()
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if a.ready.Load() {
				return false
			}
			if !a.closing.CompareAndSwap(false, true) {
				return true
			}
			go func() { a.controller.Close(); a.ready.Store(true); wr.Quit(ctx) }()
			return true
		},
		OnShutdown: func(context.Context) { a.controller.Close(); onGUIShutdown(a.Current()); close(a.stopped) },
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
