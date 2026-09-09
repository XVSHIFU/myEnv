package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestEnvironmentConfigurationNextAction(t *testing.T) {
	for _, variable := range []string{"MYENV_NODE_MIRROR", "MYENV_UV_MIRROR", "MYENV_PYTHON_MIRROR", "SSL_CERT_FILE"} {
		t.Setenv(variable, "")
	}
	for _, variable := range []string{"MYENV_NODE_MIRROR", "MYENV_UV_MIRROR", "MYENV_PYTHON_MIRROR", "SSL_CERT_FILE"} {
		t.Run(variable, func(t *testing.T) {
			t.Setenv(variable, "invalid-secret-value")
			for _, args := range [][]string{{"sync", "--json", "--no-input"}, {"clean", "--cache", "uv", "--json"}} {
				namespace := t.TempDir()
				var out, diagnostic bytes.Buffer
				code := execute(args, bytes.NewReader(nil), &out, &diagnostic, "test", namespace)
				var response result
				if err := json.Unmarshal(out.Bytes(), &response); err != nil || code == 0 || response.OK || response.Changed || response.Error == nil {
					t.Fatalf("failure: %d %s %v", code, out.String(), err)
				}
				if !strings.Contains(response.Error.NextAction, variable) || strings.Contains(out.String(), "invalid-secret-value") {
					t.Fatalf("environment diagnosis: %s", out.String())
				}
				entries, err := os.ReadDir(namespace)
				if err != nil || len(entries) != 0 {
					t.Fatalf("invalid config created state: %v %v", entries, err)
				}
			}
		})
	}
}
