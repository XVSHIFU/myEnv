package cli

import (
	"context"
	"golang.org/x/sys/windows"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"
)

func TestWindowsGUIConsoleHoldsResult(t *testing.T) {
	project := os.Getenv("MYENV_TUI_NATIVE_PROJECT")
	if project == "" {
		t.Skip("requires retained isolated project")
	}
	if !privatePromptConsole(t) {
		return
	}
	input, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	result := make(chan int, 1)
	go func() {
		result <- executeContext(context.Background(), []string{"-C", project, "run", "--hold-console", os.Getenv("COMSPEC"), "/d", "/c", "exit", "7"}, input, output, output, "test", "")
	}()
	found := false
	limit := time.Now().Add(6 * time.Second)
	for time.Now().Before(limit) {
		select {
		case code := <-result:
			t.Fatalf("closed before Enter: %d", code)
		default:
		}
		buf := make([]uint16, 16000)
		var n uint32
		kernel.NewProc("ReadConsoleOutputCharacterW").Call(output.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, uintptr(unsafe.Pointer(&n)))
		if strings.Contains(string(utf16.Decode(buf[:n])), "退出码：7") {
			found = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !found {
		t.Fatal("no visible exit result")
	}
	time.Sleep(150 * time.Millisecond)
	select {
	case <-result:
		t.Fatal("did not hold output")
	default:
	}
	type key struct {
		Kind, Pad                   uint16
		Down                        int32
		Repeat, Virtual, Scan, Char uint16
		Control                     uint32
	}
	k := key{Kind: 1, Down: 1, Repeat: 1, Virtual: 13, Char: 13}
	var n uint32
	kernel.NewProc("WriteConsoleInputW").Call(input.Fd(), uintptr(unsafe.Pointer(&k)), 1, uintptr(unsafe.Pointer(&n)))
	select {
	case code := <-result:
		if code != 7 {
			t.Fatalf("exit code %d", code)
		}
	case <-time.After(time.Second):
		t.Fatal("Enter did not close")
	}
}
