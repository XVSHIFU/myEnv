package runner

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Platform identifies only the first-release runtime targets. WSL and musl
// must not silently inherit the native Linux/glibc artifact selection.
func Platform() (string, error) {
	return identifyPlatform(runtime.GOOS, runtime.GOARCH, os.ReadFile, func(path string) bool {
		info, err := os.Stat(path)
		return err == nil && info.Mode().IsRegular()
	})
}

func identifyPlatform(goos, arch string, read func(string) ([]byte, error), exists func(string) bool) (string, error) {
	switch goos + "/" + arch {
	case "windows/amd64":
		return "windows-amd64", nil
	case "darwin/arm64":
		return "darwin-arm64", nil
	case "linux/amd64":
		release, err := read("/proc/sys/kernel/osrelease")
		if err != nil {
			return "", fmt.Errorf("identify Linux kernel: %w", err)
		}
		if strings.Contains(strings.ToLower(string(release)), "microsoft") {
			return "", fmt.Errorf("WSL is detected and is not a validated first-release target")
		}
		if exists("/lib/ld-musl-x86_64.so.1") {
			return "", fmt.Errorf("musl Linux is not supported")
		}
		if !exists("/lib64/ld-linux-x86-64.so.2") {
			return "", fmt.Errorf("Linux x86_64 requires the glibc loader /lib64/ld-linux-x86-64.so.2")
		}
		return "linux-amd64-glibc", nil
	default:
		return "", fmt.Errorf("unsupported platform %s/%s", goos, arch)
	}
}
