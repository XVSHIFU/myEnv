package cli

import (
	"context"
	"os"
	"testing"
	"time"
)

// Driven by a real PTY against a retained isolated SDK project. The inherited
// deadline must survive Tea shutdown and reach the independent supervisor.
func TestTUINativeDeadlineHelper(t *testing.T) {
	directory := os.Getenv("MYENV_TUI_NATIVE_PROJECT")
	if directory == "" {
		t.Skip("requires isolated project and external PTY driver")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	os.Exit(executeContext(ctx, []string{"-C", directory, "tui"}, os.Stdin, os.Stdout, os.Stderr, "native-test", ""))
}
