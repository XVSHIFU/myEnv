package cli

import (
	"fmt"
	"strings"
)

func providerSelection(selection, provider string) (string, error) {
	if provider == "" {
		return selection, nil
	}
	tool, version, ok := strings.Cut(selection, "@")
	if !ok || tool != "python" || (provider != "python.org" && provider != "astral") || strings.Contains(version, "/") {
		return "", fmt.Errorf("--provider requires python@version and python.org or astral")
	}
	return tool + "@" + provider + "/" + version, nil
}
