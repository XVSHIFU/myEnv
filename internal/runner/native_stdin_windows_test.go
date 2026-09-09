package runner

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWindowsNativeUnreadStdin(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	code, err := Execute(ctx, Process{Executable: prepared.Executable, Args: []string{"-e", "process.exit(0)"}, Stdin: strings.NewReader(strings.Repeat("x", 1<<20))})
	if err != nil || code != 0 || ctx.Err() != nil {
		t.Fatalf("unread stdin: %d %v context=%v", code, err, ctx.Err())
	}
}
