package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessNodeArguments(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node runtime")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err = json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "args.js")
	source := `process.stdout.write(JSON.stringify(process.argv.slice(2))); process.stderr.write(require('fs').readFileSync(0)); process.exit(17)`
	if err = os.WriteFile(script, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"space value", "--json", "quote\"value", "中文"}
	var out, diagnostic bytes.Buffer
	code, err := Execute(context.Background(), Process{Executable: prepared.Executable, Args: append([]string{script}, args...), Directory: dir, Environment: os.Environ(), Stdin: bytes.NewBufferString("input bytes"), Stdout: &out, Stderr: &diagnostic})
	if err != nil || code != 17 {
		t.Fatalf("exit %d: %v", code, err)
	}
	var got []string
	if err = json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(args) {
		t.Fatalf("args %v", got)
	}
	for i := range args {
		if got[i] != args[i] {
			t.Fatalf("argv changed: %v", got)
		}
	}
	if diagnostic.String() != "input bytes" {
		t.Fatalf("stdio changed: %s", diagnostic.String())
	}
}
