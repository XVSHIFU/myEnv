package config

import (
	"strings"
	"testing"
)

func TestLockRequiresCanonicalExactVersion(t *testing.T) {
	for _, version := range []string{"22", "22.1", "v22.1.0", "022.1.0", ">=22.1.0", "22.1.0"} {
		lock := &Lock{Schema: 1, ConfigDigest: strings.Repeat("a", 64), Platforms: map[string]PlatformLock{"windows-amd64": {Tools: map[string]RuntimeLock{"node": {Version: version, Backend: "node-official-v1", Evidence: "version"}}}}}
		err := lock.Validate()
		if version == "22.1.0" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Errorf("accepted noncanonical exact version %q", version)
		}
	}
}
