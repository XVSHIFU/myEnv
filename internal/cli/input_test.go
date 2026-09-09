package cli

import (
	"bytes"
	"errors"
	"myenv/internal/config"
	"testing"
)

func TestVersionPrompt(t *testing.T) {
	var out bytes.Buffer
	prompt := versionPrompt(bytes.NewBufferString("node@22\n"), &out)
	tool, version, err := prompt("", "missing runtime")
	if err != nil || tool != "node" || version != "22" {
		t.Fatalf("%s %s %v", tool, version, err)
	}
	_, _, err = prompt("python", "missing Python")
	var needs *config.NeedsInput
	if !errors.As(err, &needs) {
		t.Fatalf("EOF did not cancel: %v", err)
	}
	if terminalInput(bytes.NewReader(nil), &out) {
		t.Fatal("pipe treated as terminal")
	}
}

func TestBuildPrompt(t *testing.T) {
	for _, answer := range []string{"yes\n", "Y\n", "no\n", "\n", ""} {
		var output bytes.Buffer
		allowed, err := buildPrompt(bytes.NewBufferString(answer), &output)("Project metadata may run build code.")
		if err != nil || allowed != (answer == "yes\n" || answer == "Y\n") {
			t.Fatalf("answer %q: %t %v", answer, allowed, err)
		}
	}
}
