package runner

import "testing"

func TestPlatformIdentification(t *testing.T) {
	for _, tc := range []struct {
		os, arch, kernel string
		musl, glibc      bool
		want             string
	}{
		{"windows", "amd64", "", false, false, "windows-amd64"},
		{"darwin", "arm64", "", false, false, "darwin-arm64"},
		{"linux", "amd64", "6.1", false, true, "linux-amd64-glibc"},
		{"linux", "amd64", "6.1-microsoft-standard-WSL2", false, true, ""},
		{"linux", "amd64", "6.1", true, true, ""},
		{"linux", "amd64", "6.1", false, false, ""},
		{"linux", "arm64", "6.1", false, true, ""},
	} {
		got, err := identifyPlatform(tc.os, tc.arch, func(string) ([]byte, error) { return []byte(tc.kernel), nil }, func(p string) bool {
			if p == "/lib/ld-musl-x86_64.so.1" {
				return tc.musl
			}
			return tc.glibc
		})
		if got != tc.want || (err != nil) != (tc.want == "") {
			t.Errorf("%+v: %q %v", tc, got, err)
		}
	}
}
