package config

import (
	"regexp"
	"strings"
)

func SupportedTool(tool string) bool {
	switch tool {
	case "node", "python", "java", "go", "rust":
		return true
	}
	return false
}

// SDK selectors retain the upstream release spelling, including Java build IDs.
// Prefixes select a version family; catalog-selected full names select exactly.
var sdkNumeric = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+){0,3}(?:\+[0-9]+)?$`)
var javaLegacy = regexp.MustCompile(`^(?:jdk)?8u[0-9]+(?:-b[0-9]+)?$`)
var goPreview = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:beta|rc)[0-9]+$`)
var javaPreview = regexp.MustCompile(`^jdk-[0-9]+(?:\.[0-9]+){0,3}\+[0-9]+-ea(?:-beta)?$`)
var javaDatedPreview = regexp.MustCompile(`^jdk[0-9]+u-[0-9]{4}-[0-9]{2}-[0-9]{2}-[0-9]{2}-[0-9]{2}-beta$`)
var rustChannel = regexp.MustCompile(`^(?:beta|nightly)(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)
var javaFamilyAlias = regexp.MustCompile(`^(?:(?:java|jdk)-?)?(1\.8|[0-9]{1,2})$`)

// NormalizeJavaSelector only maps conventional family names. Exact upstream
// release/build IDs retain their spelling and other tools never use this map.
func NormalizeJavaSelector(text string) string {
	prefix := ""
	if strings.HasPrefix(text, "=") {
		prefix, text = "=", strings.TrimPrefix(text, "=")
	}
	if m := javaFamilyAlias.FindStringSubmatch(strings.ToLower(text)); m != nil {
		text = m[1]
		if text == "1.8" {
			text = "8"
		}
	}
	return prefix + text
}

func sdkSelector(tool, text string) bool {
	text = strings.TrimPrefix(text, "=")
	if len(text) > 128 {
		return false
	}
	if tool == "go" && goPreview.MatchString(text) {
		return true
	}
	if tool == "rust" && rustChannel.MatchString(text) {
		return true
	}
	if tool == "java" && (javaPreview.MatchString(text) || javaDatedPreview.MatchString(text)) {
		return true
	}
	if tool == "java" {
		return sdkNumeric.MatchString(strings.TrimPrefix(text, "jdk-")) || javaLegacy.MatchString(text)
	}
	return sdkNumeric.MatchString(text) && !strings.Contains(text, "+") && strings.Count(text, ".") <= 2
}
func normalizeSDKVersion(text string) string {
	return strings.TrimPrefix(strings.TrimPrefix(text, "jdk"), "-")
}

func sdkContains(selector, version string) bool {
	if selector == "nightly" || selector == "beta" {
		return version == selector || strings.HasPrefix(version, selector+"-")
	}
	if strings.HasPrefix(selector, "=") {
		return normalizeSDKVersion(strings.TrimPrefix(selector, "=")) == normalizeSDKVersion(version)
	}
	a, b := normalizeSDKVersion(selector), normalizeSDKVersion(version)
	if javaLegacy.MatchString(a) && javaLegacy.MatchString(b) && !strings.Contains(a, "-b") && strings.HasPrefix(b, a+"-b") {
		return true
	}
	return a == b || strings.HasPrefix(b, a+".") || strings.HasPrefix(b, a+"+") || strings.HasPrefix(b, a+"u")
}
