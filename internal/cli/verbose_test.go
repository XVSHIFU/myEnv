package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVerboseStatusKeepsJSON(t *testing.T) {
	for _, verbose := range []bool{false, true} {
		var out, diagnostic bytes.Buffer
		root := t.TempDir()
		args := []string{"-C", root, "--json"}
		if verbose {
			args = append(args, "--verbose")
		}
		code := Execute(args, bytes.NewReader(nil), &out, &diagnostic, "test")
		var response result
		if err := json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if code != 0 || !response.OK || response.Changed {
			t.Fatalf("exit %d: %s", code, out.String())
		}
		if verbose {
			if !strings.Contains(diagnostic.String(), "command=myenv platform=") || !strings.Contains(diagnostic.String(), "command=myenv elapsed=") {
				t.Fatalf("missing diagnostics: %s", diagnostic.String())
			}
			if strings.Contains(diagnostic.String(), root) {
				t.Fatal("diagnostics exposed argument path")
			}
		} else if diagnostic.Len() != 0 {
			t.Fatalf("unexpected diagnostics: %s", diagnostic.String())
		}
	}
}

func TestVerboseRunRetainedNode(t *testing.T) {
	root, _ := prepareRetainedNodeRun(t)
	var out, diagnostic bytes.Buffer
	secret := "literal-private-argument"
	code := Execute([]string{"-C", root, "--verbose", "run", "node", "-e", "process.stdout.write(process.argv[1]);process.exit(19)", "--", secret}, bytes.NewReader(nil), &out, &diagnostic, "test")
	if code != 19 || out.String() != secret {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	if !strings.Contains(diagnostic.String(), "command=run platform=") || !strings.Contains(diagnostic.String(), "command=run elapsed=") {
		t.Fatalf("missing run diagnostics: %s", diagnostic.String())
	}
	if strings.Contains(diagnostic.String(), secret) || strings.Contains(diagnostic.String(), root) {
		t.Fatalf("run diagnostics exposed arguments: %s", diagnostic.String())
	}
}
