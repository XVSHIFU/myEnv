//go:build windows

package cli

import "golang.org/x/sys/windows"

var userUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

func systemChinese() bool {
	id, _, _ := userUILanguage.Call()
	return id&0x3ff == 0x04
}
