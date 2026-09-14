package cli

import (
	"path/filepath"
	"strings"
)

// A published PythonExecutable is the generation's venv entry, including
// runtime-only profiles. Avoid inheriting another interpreter's PYTHONHOME;
// explicit project variables still take precedence in the final overlay.
func pythonRunEnvironment(parent []string, overrides map[string]string, executable string, windows bool) []string {
	if executable == "" {
		return parent
	}
	equal := func(a, b string) bool {
		return a == b || windows && strings.EqualFold(a, b)
	}
	filtered := make([]string, 0, len(parent))
	for _, entry := range parent {
		name, _, _ := strings.Cut(entry, "=")
		if !equal(name, "PYTHONHOME") {
			filtered = append(filtered, entry)
		}
	}
	for name := range overrides {
		if equal(name, "VIRTUAL_ENV") {
			return filtered
		}
	}
	overrides["VIRTUAL_ENV"] = filepath.Dir(filepath.Dir(executable))
	return filtered
}
