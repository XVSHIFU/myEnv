package runner

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"strings"
	"testing"
)

func TestBackgroundConsoleChild(t *testing.T) {
	if os.Getenv("MYENV_TEST_CONSOLE_CHILD") != "1" {
		return
	}
	handle, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
	fmt.Printf("console=%d\n", handle)
	os.Exit(0)
}
func TestBackgroundProcessHasNoConsole(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	code, err := Execute(BackgroundContext(context.Background()), Process{Executable: exe, Args: []string{"-test.run=^TestBackgroundConsoleChild$"}, Environment: append(os.Environ(), "MYENV_TEST_CONSOLE_CHILD=1"), Stdout: &output, Stderr: &output})
	if err != nil || code != 0 || !strings.Contains(output.String(), "console=0") {
		t.Fatalf("%d %v %s", code, err, &output)
	}
}
