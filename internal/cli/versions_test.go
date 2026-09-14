package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionsRejectsInvalidOptionsBeforeClient(t *testing.T) {
	// Invalid TLS configuration would fail before the intended usage error if
	// option validation happened after constructing the download client.
	t.Setenv("SSL_CERT_FILE", filepath.Join(t.TempDir(), "missing-ca.pem"))
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"ruby"}, "unsupported catalog tool"},
		{[]string{"node", "--major", "0"}, "--major"},
		{[]string{"java", "--major", "0"}, "--major"},
		{[]string{"node", "--provider", "python.org"}, "--provider"},
		{[]string{"rust", "--channel", "nightly", "--provider", "python.org"}, "--provider"},
		{[]string{"rust", "--date", "2026-09-01"}, "--channel"},
		{[]string{"rust", "--channel", "nightly", "--date", "2026-02-30"}, "--date requires"},
		{[]string{"rust", "--channel", "nightly", "--date="}, "--date requires"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			args := append([]string{"versions", "--json"}, tc.args...)
			var out, diagnostic bytes.Buffer
			code := execute(args, strings.NewReader(""), &out, &diagnostic, "test", t.TempDir())
			var response result
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatalf("invalid JSON: %s (%v)", &out, err)
			}
			if code != 2 || response.OK || response.Changed || response.Error == nil || response.Error.Code != "USAGE_ERROR" || !strings.Contains(response.Error.Message, tc.want) {
				t.Fatalf("exit %d: %s %s", code, &out, &diagnostic)
			}
		})
	}
}
