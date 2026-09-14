package backend

import (
	"regexp"
	"strings"
)

var registrySemver = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var registryPEP440 = regexp.MustCompile(`(?i)^v?(?:([0-9]+)!)?([0-9]+(?:\.[0-9]+)*)(?:[-_.]?(a|b|c|rc|alpha|beta|pre|preview)[-_.]?([0-9]+)?)?(?:(?:-([0-9]+))|(?:[-_.]?(post|rev|r)[-_.]?([0-9]+)?))?(?:[-_.]?(dev)[-_.]?([0-9]+)?)?(?:\+([a-z0-9]+(?:[-_.][a-z0-9]+)*))?$`)

type registryVersionSortKey struct {
	valid, preview        bool
	epoch                 string
	release               []string
	pre                   []string // SemVer identifiers, if present.
	preRank               int      // PEP440: development-only=-1, a=0, b=1, rc=2, final=3.
	preNumber             string
	post, dev             bool
	postNumber, devNumber string
	local                 []string
}

// These keys order registry labels; they do not replace a native manager's
// dependency constraint solver or Python/platform compatibility checks.
func registryVersionKey(eco, version string) registryVersionSortKey {
	if len(version) == 0 || len(version) > 128 {
		return registryVersionSortKey{}
	}
	if eco == "node" {
		match := registrySemver.FindStringSubmatch(version)
		if match == nil {
			return registryVersionSortKey{}
		}
		key := registryVersionSortKey{valid: true, release: match[1:4], preview: match[4] != ""}
		if key.preview {
			key.pre = strings.Split(match[4], ".")
			for _, part := range key.pre {
				if registryDigits(part) && len(part) > 1 && part[0] == '0' {
					return registryVersionSortKey{}
				}
			}
		}
		return key
	}
	match := registryPEP440.FindStringSubmatch(version)
	if match == nil {
		return registryVersionSortKey{}
	}
	key := registryVersionSortKey{valid: true, epoch: match[1], release: strings.Split(match[2], "."), preRank: 3, preNumber: match[4]}
	pre := strings.ToLower(match[3])
	key.post = match[5] != "" || match[6] != ""
	key.postNumber = match[7]
	if match[5] != "" {
		key.postNumber = match[5]
	}
	key.dev = match[8] != ""
	key.devNumber = match[9]
	key.preview = pre != "" || key.dev
	switch pre {
	case "a", "alpha":
		key.preRank = 0
	case "b", "beta":
		key.preRank = 1
	case "c", "rc", "pre", "preview":
		key.preRank = 2
	default:
		if key.dev && !key.post {
			key.preRank = -1
		}
	}
	if match[10] != "" {
		key.local = strings.FieldsFunc(strings.ToLower(match[10]), func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	}
	return key
}

func compareRegistryVersions(eco, a, b string) int {
	return compareRegistryVersionKeys(eco, registryVersionKey(eco, a), registryVersionKey(eco, b))
}

func compareRegistryVersionKeys(eco string, a, b registryVersionSortKey) int {
	if a.valid != b.valid {
		if a.valid {
			return 1
		}
		return -1
	}
	if n := registryCompareNumber(a.epoch, b.epoch); n != 0 {
		return n
	}
	length := max(len(a.release), len(b.release))
	for i := 0; i < length; i++ {
		x, y := "", ""
		if i < len(a.release) {
			x = a.release[i]
		}
		if i < len(b.release) {
			y = b.release[i]
		}
		if n := registryCompareNumber(x, y); n != 0 {
			return n
		}
	}
	if eco == "node" {
		if len(a.pre) == 0 && len(b.pre) > 0 {
			return 1
		}
		if len(b.pre) == 0 && len(a.pre) > 0 {
			return -1
		}
		return registryCompareSegments(a.pre, b.pre, false)
	}
	if a.preRank != b.preRank {
		if a.preRank > b.preRank {
			return 1
		}
		return -1
	}
	if n := registryCompareNumber(a.preNumber, b.preNumber); n != 0 {
		return n
	}
	if a.post != b.post {
		if a.post {
			return 1
		}
		return -1
	}
	if n := registryCompareNumber(a.postNumber, b.postNumber); n != 0 {
		return n
	}
	if a.dev != b.dev {
		if a.dev {
			return -1
		}
		return 1
	}
	if n := registryCompareNumber(a.devNumber, b.devNumber); n != 0 {
		return n
	}
	return registryCompareSegments(a.local, b.local, true)
}

func registryCompareNumber(a, b string) int {
	a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		if len(a) > len(b) {
			return 1
		}
		return -1
	}
	return strings.Compare(a, b)
}

func registryDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func registryCompareSegments(a, b []string, numbersHigher bool) int {
	for i := 0; i < min(len(a), len(b)); i++ {
		aNumber, bNumber := registryDigits(a[i]), registryDigits(b[i])
		if aNumber && bNumber {
			if n := registryCompareNumber(a[i], b[i]); n != 0 {
				return n
			}
			continue
		}
		if aNumber != bNumber {
			if aNumber == numbersHigher {
				return 1
			}
			return -1
		}
		if n := strings.Compare(a[i], b[i]); n != 0 {
			return n
		}
	}
	if len(a) > len(b) {
		return 1
	}
	if len(a) < len(b) {
		return -1
	}
	return 0
}
