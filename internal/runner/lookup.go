package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Lookup searches the child's environment without modifying process-global PATH.
func Lookup(command, cwd string, env []string, windows bool) (string, error) {
	value := func(key string) string {
		for i := len(env) - 1; i >= 0; i-- {
			name, v, ok := strings.Cut(env[i], "=")
			if ok && (name == key || windows && strings.EqualFold(name, key)) {
				return v
			}
		}
		return ""
	}
	extensions := []string{""}
	if windows && filepath.Ext(command) == "" {
		extensions = nil
		ext := value("PATHEXT")
		if ext == "" {
			ext = ".COM;.EXE;.BAT;.CMD"
		}
		for _, e := range strings.Split(ext, ";") {
			if strings.HasPrefix(e, ".") && !strings.ContainsAny(e, "/\\:") {
				extensions = append(extensions, e)
			}
		}
	}
	check := func(base string) string {
		for _, ext := range extensions {
			candidate := base + ext
			info, err := os.Stat(candidate)
			if err == nil && info.Mode().IsRegular() && (windows || info.Mode().Perm()&0111 != 0) {
				return candidate
			}
		}
		return ""
	}
	if filepath.IsAbs(command) || strings.ContainsAny(command, "/\\") {
		candidate := command
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(cwd, candidate)
		}
		if found := check(candidate); found != "" {
			return filepath.Abs(found)
		}
	} else {
		separator := ":"
		if windows {
			separator = ";"
		}
		for _, directory := range strings.Split(value("PATH"), separator) {
			if directory == "" {
				continue
			}
			if !filepath.IsAbs(directory) {
				directory = filepath.Join(cwd, directory)
			}
			if found := check(filepath.Join(directory, command)); found != "" {
				return filepath.Abs(found)
			}
		}
	}
	return "", fmt.Errorf("ENV_NOT_READY: executable %q not found in applied PATH", command)
}
