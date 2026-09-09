package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLockDuplicateKeys(t *testing.T) {
	for _, document := range []string{
		`{"schema":1,"schema":1}`,
		`{"platforms":{"windows-amd64":{},"windows-amd64":{}}}`,
		`{"platforms":{"windows-amd64":{"tools":{"node":{"version":"22.1.0","version":"22.2.0"}}}}}`,
		`{"schema":1,"\u0073chema":1}`,
	} {
		path := filepath.Join(t.TempDir(), "myenv.lock")
		if err := os.WriteFile(path, []byte(document), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadLock(path); err == nil || !strings.Contains(err.Error(), "duplicate JSON key") {
			t.Fatalf("accepted ambiguous lock %s: %v", document, err)
		}
	}
}

func TestJSONKeyBoundaries(t *testing.T) {
	for _, document := range []string{`{"a":{"version":1},"b":{"version":2}}`, `{"a":[{"x":1},{"x":2}]}`} {
		if err := checkJSONKeys([]byte(document)); err != nil {
			t.Fatal(err)
		}
	}
	for _, document := range []string{`{"a":1} {}`, `{"a":`, `{"a":[{"x":1,"x":2}]}`, strings.Repeat("[", 66) + "0" + strings.Repeat("]", 66)} {
		if err := checkJSONKeys([]byte(document)); err == nil {
			t.Fatal("accepted malformed or ambiguous JSON", document)
		}
	}
}

func TestLockCanonicalKeys(t *testing.T) {
	for _, document := range []string{
		`{"schema":1,"Schema":2}`,
		`{"platforms":{"windows-amd64":{"tools":{"node":{"Version":"22.1.0"}}}}}`,
		`{"platforms":{"windows-amd64":{"python":{"UV_LOCK_SHA256":"x"}}}}`,
		`{"\u0053chema":1}`,
		`{"ſchema":1}`,
	} {
		path := filepath.Join(t.TempDir(), "myenv.lock")
		if err := os.WriteFile(path, []byte(document), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadLock(path); err == nil || !strings.Contains(err.Error(), "lowercase ASCII") {
			t.Fatalf("accepted noncanonical key: %s %v", document, err)
		}
	}
	if err := checkJSONKeys([]byte(`{"project":"项目/CaseSensitive","url":"https://example.test/Case","windows-amd64":{}}`)); err != nil {
		t.Fatal("changed value semantics", err)
	}
}
