package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version is a stable runtime release. Pre-release selection is not implicit.
type Version [3]int
type interval struct{ low, high Version } // half-open; high is exclusive
type Constraint struct {
	spans []interval
	sdk   string
}

var infinity = Version{1 << 30, 0, 0}
var versionToken = regexp.MustCompile(`^(>=|<=|!=|==|~=|>|<|=|\^|~)?\s*(v?[0-9]+(?:\.(?:[0-9]+|[xX*])){0,2}|[xX*])`)
var hyphenRange = regexp.MustCompile(`^(v?[0-9]+(?:\.[0-9]+){0,2})\s+-\s+(v?[0-9]+(?:\.[0-9]+){0,2})(?:\s*,\s*|\s+|$)`)

func compare(a, b Version) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
func parseRelease(s string) (Version, int, error) {
	var v Version
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) > 3 {
		return v, 0, fmt.Errorf("invalid version %q", s)
	}
	count := 0
	wild := false
	for i, p := range parts {
		if p == "*" || strings.EqualFold(p, "x") {
			wild = true
			continue
		}
		if wild {
			return v, 0, fmt.Errorf("invalid wildcard %q", s)
		}
		n, e := strconv.Atoi(p)
		if e != nil || n < 0 || n >= 1<<29 {
			return v, 0, fmt.Errorf("invalid version %q", s)
		}
		v[i] = n
		count++
	}
	return v, count, nil
}
func next(v Version, component int) Version {
	v[component]++
	for i := component + 1; i < 3; i++ {
		v[i] = 0
	}
	return v
}
func intersect(a, b []interval) []interval {
	var out []interval
	for _, x := range a {
		for _, y := range b {
			lo, hi := x.low, x.high
			if compare(y.low, lo) > 0 {
				lo = y.low
			}
			if compare(y.high, hi) < 0 {
				hi = y.high
			}
			if compare(lo, hi) < 0 {
				out = append(out, interval{lo, hi})
			}
		}
	}
	return out
}

// ParseConstraint accepts stable numeric selectors and native common range forms.
// Unsupported syntax is rejected, never handed to a shell or guessed as a version.
func ParseConstraint(tool, text string) (Constraint, error) {
	if tool == "node" && NodeChannel(text) != "" {
		return Constraint{sdk: "=" + text}, nil
	}
	if tool == "python" {
		provider, version := PythonProvider(text)
		if provider != "astral" && provider != "python.org" {
			return Constraint{}, fmt.Errorf("unsupported Python provider %q", provider)
		}
		text = version
		if pythonPreview.MatchString(text) {
			return Constraint{sdk: "=" + text}, nil
		}
	}
	if tool == "java" || tool == "go" || tool == "rust" {
		if !sdkSelector(tool, text) {
			return Constraint{}, fmt.Errorf("invalid %s version %q", tool, text)
		}
		return Constraint{sdk: text}, nil
	}
	fail := func() (Constraint, error) {
		return Constraint{}, fmt.Errorf("invalid %s version constraint %q", tool, text)
	}
	if tool != "node" && tool != "python" || len(text) > 1024 || strings.ContainsAny(text, "\r\n\x00") {
		return fail()
	}
	var result Constraint
	for _, branch := range strings.Split(strings.TrimSpace(text), "||") {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			return fail()
		}
		spans := []interval{{Version{}, infinity}}
		for branch != "" {
			if tool == "node" {
				if match := hyphenRange.FindStringSubmatch(branch); match != nil {
					if strings.HasSuffix(strings.TrimSpace(match[0]), ",") && len(match[0]) == len(branch) {
						return fail()
					}
					low, _, e1 := parseRelease(match[1])
					high, count, e2 := parseRelease(match[2])
					if e1 != nil || e2 != nil {
						return fail()
					}
					spans = intersect(spans, []interval{{low, next(high, count-1)}})
					branch = strings.TrimSpace(branch[len(match[0]):])
					continue
				}
			}
			token := versionToken.FindStringSubmatch(branch)
			if token == nil {
				return fail()
			}
			op, raw := token[1], token[2]
			if tool == "python" && strings.ContainsAny(raw, "*xX") && op != "" && op != "==" && op != "!=" {
				return fail()
			}
			v, n, err := parseRelease(raw)
			if err != nil {
				return fail()
			}
			hi := infinity
			if n > 0 {
				hi = next(v, n-1)
			}
			if tool == "python" && op != "" && !strings.ContainsAny(raw, "*xX") {
				hi = next(v, 2)
			}
			var condition []interval
			switch op {
			case "", "=", "==":
				condition = []interval{{v, hi}}
			case ">=":
				condition = []interval{{v, infinity}}
			case ">":
				if n == 0 {
					return fail()
				}
				condition = []interval{{hi, infinity}}
			case "<":
				condition = []interval{{Version{}, v}}
			case "<=":
				condition = []interval{{Version{}, hi}}
			case "!=":
				condition = []interval{{Version{}, v}, {hi, infinity}}
			case "~":
				if tool != "node" || n == 0 {
					return fail()
				}
				index := 0
				if n >= 2 {
					index = 1
				}
				condition = []interval{{v, next(v, index)}}
			case "^":
				if tool != "node" || n == 0 {
					return fail()
				}
				index := 0
				for index < n-1 && v[index] == 0 {
					index++
				}
				condition = []interval{{v, next(v, index)}}
			case "~=":
				if tool != "python" || n < 2 {
					return fail()
				}
				condition = []interval{{v, next(v, n-2)}}
			default:
				return fail()
			}
			spans = intersect(spans, condition)
			rest := branch[len(token[0]):]
			if rest != "" && rest[0] != ' ' && rest[0] != '\t' && rest[0] != ',' {
				return fail()
			}
			branch = strings.TrimSpace(rest)
			if strings.HasPrefix(branch, ",") {
				branch = strings.TrimSpace(branch[1:])
				if branch == "" {
					return fail()
				}
			}
		}
		result.spans = append(result.spans, spans...)
	}
	if len(result.spans) == 0 {
		return Constraint{}, fmt.Errorf("%s version constraint %q has no matching stable version", tool, text)
	}
	return result, nil
}

func (c Constraint) Contains(release string) bool {
	if c.sdk != "" {
		return sdkContains(c.sdk, release)
	}
	v, n, err := parseRelease(release)
	if err != nil || n != 3 {
		return false
	}
	for _, s := range c.spans {
		if compare(v, s.low) >= 0 && compare(v, s.high) < 0 {
			return true
		}
	}
	return false
}

// MergeConstraints preserves both inputs when their intersection is nonempty.
func MergeConstraints(tool, a, b string) (string, error) {
	left, err := ParseConstraint(tool, a)
	if err != nil {
		return "", err
	}
	right, err := ParseConstraint(tool, b)
	if err != nil {
		return "", err
	}
	if left.sdk != "" {
		if left.Contains(strings.TrimPrefix(b, "=")) {
			return b, nil
		}
		if right.Contains(strings.TrimPrefix(a, "=")) {
			return a, nil
		}
		return "", fmt.Errorf("incompatible constraints %q and %q", a, b)
	}
	if len(intersect(left.spans, right.spans)) == 0 {
		return "", fmt.Errorf("incompatible constraints %q and %q", a, b)
	}
	if a == b {
		return a, nil
	}
	// Distribute unions, avoiding parentheses unsupported by native range grammars.
	var parts []string
	for _, x := range strings.Split(a, "||") {
		for _, y := range strings.Split(b, "||") {
			parts = append(parts, strings.TrimSpace(x)+", "+strings.TrimSpace(y))
		}
	}
	merged := strings.Join(parts, " || ")
	if _, err := ParseConstraint(tool, merged); err != nil {
		return "", err
	}
	return merged, nil
}
