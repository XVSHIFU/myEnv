package core

import (
	"encoding/hex"
	"strconv"
	"strings"
)

func validLinuxOwnerIdentity(identity string) bool {
	parts := strings.Split(identity, ":")
	if len(parts) != 3 || parts[0] != "linux" || len(parts[1]) != 36 {
		return false
	}
	boot := parts[1]
	for _, i := range []int{8, 13, 18, 23} {
		if boot[i] != '-' {
			return false
		}
	}
	rawBoot := strings.ReplaceAll(boot, "-", "")
	decoded, err := hex.DecodeString(rawBoot)
	if err != nil || len(decoded) != 16 || hex.EncodeToString(decoded) != rawBoot {
		return false
	}
	start, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil || strconv.FormatUint(start, 10) != parts[2] {
		return false
	}
	return true
}
