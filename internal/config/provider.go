package config

import (
	"regexp"
	"strings"
)

var pythonPreview = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:a|b|rc)[0-9]+$`)
var nodePreview = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+-(nightly[0-9]{8}[0-9a-f]+|rc\.[0-9]+|v8-canary[0-9]{8}[0-9a-f]+)$`)

func NodeChannel(version string) string {
	if !nodePreview.MatchString(version) {
		return ""
	}
	if strings.Contains(version, "-nightly") {
		return "nightly"
	}
	if strings.Contains(version, "-v8-canary") {
		return "v8-canary"
	}
	return "rc"
}

func PythonProvider(selector string) (string, string) {
	if provider, version, ok := strings.Cut(selector, "/"); ok {
		return provider, version
	}
	return "astral", selector
}

func IsPreview(tool, version string) bool {
	if tool == "node" {
		return NodeChannel(version) != ""
	}
	if tool == "python" {
		_, version = PythonProvider(version)
		return pythonPreview.MatchString(version)
	}
	if tool == "rust" {
		return strings.HasPrefix(version, "nightly") || strings.HasPrefix(version, "beta") || strings.Contains(version, "-beta") || strings.Contains(version, "-nightly")
	}
	if tool == "go" {
		return strings.Contains(version, "beta") || strings.Contains(version, "rc")
	}
	return strings.Contains(version, "-ea") || strings.Contains(version, "-beta") || strings.Contains(version, "-rc")
}
