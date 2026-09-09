package runner

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestCancelRetainedNode(t *testing.T) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Node")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct{ Executable string }
	if err = json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	code, err := Execute(ctx, Process{Executable: prepared.Executable, Args: []string{"-e", "setInterval(()=>{},1000)"}, Environment: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil || code == 0 || code < 0 {
		t.Fatalf("invalid cancellation result: %d %v", code, ctx.Err())
	}
}
