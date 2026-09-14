package backend

import (
	"strings"
	"testing"

	"myenv/internal/config"
)

func TestJavaVersionOutput(t *testing.T) {
	// Adoptium's JDK 8 metadata includes -bNN; java -version puts that build
	// on the Runtime Environment line, outside the quoted version.
	java8 := "openjdk version \"1.8.0_422\"\nOpenJDK Runtime Environment (Temurin)(build 1.8.0_422-b05)\n"
	for _, row := range []struct {
		name, release, runtimeVersion, output string
		want                                  bool
	}{
		{"java8 metadata", "jdk8u422-b05", "1.8.0_422-b05", java8, true},
		{"java8 legacy lock", "jdk8u422-b05", "", java8, true},
		{"java8 different update", "jdk8u422-b05", "1.8.0_422-b05", strings.ReplaceAll(java8, "422", "432"), false},
		{"java8 different build", "jdk8u422-b05", "1.8.0_422-b05", strings.ReplaceAll(java8, "b05", "b06"), false},
		{"java8 missing build", "jdk8u422-b05", "1.8.0_422-b05", "openjdk version \"1.8.0_422\"\n", false},
		{"java21", "jdk-21.0.2+13", "21.0.2+13", "openjdk version \"21.0.2\" 2024-01-16 LTS\n", true},
		{"java21 different update", "jdk-21.0.2+13", "21.0.2+13", "openjdk version \"21.0.3\"\n", false},
		{"java EA", "jdk-27+14-ea-beta", "27-ea+14", "openjdk version \"27-ea\"\n", true},
	} {
		t.Run(row.name, func(t *testing.T) {
			lock := config.RuntimeLock{Version: row.release, RuntimeVersion: row.runtimeVersion}
			if got := javaVersionMatches(lock, row.output); got != row.want {
				t.Fatalf("version match=%v, want %v for %q", got, row.want, row.output)
			}
		})
	}
}
