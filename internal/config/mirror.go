package config

import (
	"fmt"
	"net/url"
	"strings"
)

// MirrorBaseURL validates a transport override without echoing possible secrets.
func MirrorBaseURL(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", fmt.Errorf("must be an HTTP(S) base URL without credentials, query or fragment")
	}
	return strings.TrimRight(u.String(), "/"), nil
}
