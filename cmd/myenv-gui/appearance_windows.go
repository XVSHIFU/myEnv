//go:build windows && gui

package main

import (
	"fmt"
	"strings"

	win "github.com/wailsapp/wails/v2/pkg/options/windows"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

// SetAppearance keeps the native caption aligned with the selected app theme.
// The frontend supplies the complete title, including the project and myEnv.
func (a *App) SetAppearance(theme string, title string) error {
	if theme != "system" && theme != "light" && theme != "dark" {
		return fmt.Errorf("不支持的主题：%s", theme)
	}
	if strings.ContainsRune(title, '\x00') {
		return fmt.Errorf("窗口标题不能包含空字符")
	}
	if a.ctx == nil {
		return fmt.Errorf("窗口尚未就绪")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "myEnv"
	}
	switch theme {
	case "light":
		wr.WindowSetLightTheme(a.ctx)
	case "dark":
		wr.WindowSetDarkTheme(a.ctx)
	default:
		// Wails follows WM_SETTINGCHANGE without a polling loop or custom frame.
		wr.WindowSetSystemDefaultTheme(a.ctx)
	}
	wr.WindowSetTitle(a.ctx, title)
	return nil
}

func nativeWindowTheme() *win.ThemeSettings {
	lightChrome, lightText := win.RGB(0xf0, 0xf3, 0xee), win.RGB(0x5c, 0x6a, 0x62)
	darkChrome, darkText := win.RGB(0x1b, 0x28, 0x21), win.RGB(0xa0, 0xb1, 0xa3)
	// Wails uses DWM colours where supported and preserves the Windows frame,
	// caption buttons and high-contrast handling on all supported versions.
	return &win.ThemeSettings{
		LightModeTitleBar:          lightChrome,
		LightModeTitleBarInactive:  lightChrome,
		LightModeTitleText:         lightText,
		LightModeTitleTextInactive: lightText,
		LightModeBorder:            lightChrome,
		LightModeBorderInactive:    lightChrome,
		DarkModeTitleBar:           darkChrome,
		DarkModeTitleBarInactive:   darkChrome,
		DarkModeTitleText:          darkText,
		DarkModeTitleTextInactive:  darkText,
		DarkModeBorder:             darkChrome,
		DarkModeBorderInactive:     darkChrome,
	}
}
