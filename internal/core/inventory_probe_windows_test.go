package core

import (
	"context"
	"fmt"
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestInventoryProbeHasNoWindowsConsole(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYENV_INVENTORY_CONSOLE_TEST", "1")
	output, err := probeInventoryCommand(context.Background(), executable, []string{"-test.run=^TestInventoryProbeConsoleHelper$"}, "", 8192)
	if err != nil || output != "console=0" {
		t.Fatalf("inventory created a console: %q %v", output, err)
	}
}

func TestInventoryProbeConsoleHelper(t *testing.T) {
	if os.Getenv("MYENV_INVENTORY_CONSOLE_TEST") != "1" {
		return
	}
	handle, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
	fmt.Printf("console=%d", handle)
	os.Exit(0)
}
