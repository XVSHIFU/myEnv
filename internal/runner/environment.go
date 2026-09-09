// Package runner constructs and supervises explicit child process environments.
package runner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Environment overlays ordinary project variables and prepends managed runtime
// directories. It does not read or alter the calling process environment.
func Environment(parent []string, project map[string]string, bins []string, windows bool) ([]string, error) {
	key := func(s string) string {
		if windows {
			return strings.ToUpper(s)
		}
		return s
	}
	values := map[string]string{}
	names := map[string]string{}
	for _, entry := range parent {
		// Windows drive-current-directory entries have the form =C:=C:\work.
		offset := 0
		if strings.HasPrefix(entry, "=") {
			offset = 1
		}
		index := strings.IndexByte(entry[offset:], '=')
		if index < 0 {
			continue
		}
		index += offset
		name, value := entry[:index], entry[index+1:]
		values[key(name)] = value
		names[key(name)] = name
	}
	projectKeys := make(map[string]bool, len(project))
	for name, value := range project {
		if name == "" || strings.ContainsAny(name, "=\x00") || strings.ContainsRune(value, 0) {
			return nil, fmt.Errorf("invalid environment variable %q", name)
		}
		if projectKeys[key(name)] {
			return nil, fmt.Errorf("duplicate case-insensitive environment key %q", key(name))
		}
		projectKeys[key(name)] = true
		values[key(name)] = value
		names[key(name)] = name
	}
	separator := ":"
	if windows {
		separator = ";"
	}
	for _, bin := range bins {
		if !filepath.IsAbs(bin) || strings.Contains(bin, separator) || strings.ContainsRune(bin, 0) {
			return nil, fmt.Errorf("invalid runtime bin directory %q", bin)
		}
	}
	if len(bins) > 0 {
		path := strings.Join(bins, separator)
		if prior := values[key("PATH")]; prior != "" {
			path += separator + prior
		}
		values[key("PATH")] = path
		names[key("PATH")] = "PATH"
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, k := range keys {
		result = append(result, names[k]+"="+values[k])
	}
	return result, nil
}
