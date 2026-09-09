//go:build !windows

package cli

import (
	"os"
	"strings"
)

func systemChinese() bool {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			value = strings.ToLower(value)
			return value == "zh" || strings.HasPrefix(value, "zh_") || strings.HasPrefix(value, "zh-") || strings.HasPrefix(value, "zh.")
		}
	}
	return false
}
