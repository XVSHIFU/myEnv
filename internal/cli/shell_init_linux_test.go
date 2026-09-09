package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestBashProfileWrapperArgumentsAndExit(t *testing.T) {
	testPOSIXProfileWrapper(t, "bash", []string{"--noprofile", "--norc"}, "exit $?")
}

func TestZshProfileWrapperArgumentsAndExit(t *testing.T) {
	testPOSIXProfileWrapper(t, "zsh", []string{"-f"}, "exit $?")
}

func TestFishProfileWrapperArgumentsAndExit(t *testing.T) {
	testPOSIXProfileWrapper(t, "fish", []string{"--no-config", "--private"}, "exit $status")
}

func testPOSIXProfileWrapper(t *testing.T, shell string, options []string, exit string) {
	executable, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("requires installed %s: %v", shell, err)
	}
	root := t.TempDir()
	entry := filepath.Join(root, "probe '$ literal")
	if err := os.WriteFile(entry, []byte("#!/bin/sh\nprintf '%s\\0' \"$@\"\nexit 23\n"), 0700); err != nil {
		t.Fatal(err)
	}
	driver := filepath.Join(root, "driver.sh")
	script := profileShellScript(shell, entry, map[string]string{"node": "22"}) + "node 'a b' '' '$literal;not-code' '*.js'\n" + exit + "\n"
	if err := os.WriteFile(driver, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, append(options, driver)...)
	var diagnostic bytes.Buffer
	command.Stderr = &diagnostic
	output, err := command.Output()
	if command.ProcessState == nil || command.ProcessState.ExitCode() != 23 || diagnostic.Len() != 0 {
		t.Fatalf("wrapper: %v %s", err, diagnostic.String())
	}
	parts := bytes.Split(output, []byte{0})
	got := make([]string, len(parts))
	for i, part := range parts {
		got[i] = string(part)
	}
	want := []string{"run", "--global", "node", "a b", "", "$literal;not-code", "*.js", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%q want=%q", got, want)
	}
}
