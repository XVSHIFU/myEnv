package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"
)

func TestWindowsTUIConsoleInterrupt(t *testing.T) { testWindowsTUIConsole(t, 0, "SIGINT") }
func TestWindowsTUIConsoleBreak(t *testing.T)     { testWindowsTUIConsole(t, 1, "SIGBREAK") }
func testWindowsTUIConsole(t *testing.T, event uintptr, name string) {
	project := os.Getenv("MYENV_TUI_NATIVE_PROJECT")
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if project == "" || record == "" {
		t.Skip("requires isolated Java project and retained Node")
	}
	if !privatePromptConsole(t) {
		return
	}
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	kernel.NewProc("SetConsoleCtrlHandler").Call(0, 0)
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
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var node struct{ Executable string }
	if err = json.Unmarshal(data, &node); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "signal")
	script := fmt.Sprintf(`const fs=require('fs');let n=0;process.on('%s',()=>{n++;setTimeout(()=>{fs.writeFileSync(process.argv[1]+'.done',String(n));process.exit(23)},100)});fs.writeFileSync(process.argv[1]+'.ready','ready');setInterval(()=>{},1000)`, name)
	args, _ := json.Marshal([]string{node.Executable, "-e", script, marker})
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		screen := func() string {
			b := make([]uint16, 16000)
			var n uint32
			kernel.NewProc("ReadConsoleOutputCharacterW").Call(output.Fd(), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0, uintptr(unsafe.Pointer(&n)))
			return string(utf16.Decode(b[:n]))
		}
		wait := func(check func() bool) bool {
			for !check() {
				if ctx.Err() != nil {
					fmt.Fprintln(os.Stdout, "console timeout:", screen())
					return false
				}
				time.Sleep(20 * time.Millisecond)
			}
			return true
		}
		type key struct {
			Kind, Pad                   uint16
			Down                        int32
			Repeat, Virtual, Scan, Char uint16
			Control                     uint32
		}
		send := func(chars string, virtual uint16) error {
			for _, ch := range utf16.Encode([]rune(chars)) {
				k := key{Kind: 1, Down: 1, Repeat: 1, Virtual: virtual, Char: ch}
				var n uint32
				ok, _, e := kernel.NewProc("WriteConsoleInputW").Call(input.Fd(), uintptr(unsafe.Pointer(&k)), 1, uintptr(unsafe.Pointer(&n)))
				if ok == 0 {
					return e
				}
			}
			return nil
		}
		if !wait(func() bool { return strings.Contains(screen(), "已完成") }) {
			finished <- fmt.Errorf("TUI not ready")
			return
		}
		send("\x00", 40)
		send("\x00", 40)
		send("\x00", 40) // Java -> Go -> Rust -> Run.
		send("\r", 13)
		send("j", 0) // Signal fixture requires an exact script argv; ordinary input has separate coverage.
		send(strings.Repeat("\b", 80), 8)
		send(string(args), 0)
		send("\r", 13)
		send("\x00", 35) // End selects Run after the argument list.
		send("\r", 13)
		send("\r", 13)

		if !wait(func() bool { _, e := os.Stat(marker + ".ready"); return e == nil }) {
			finished <- fmt.Errorf("command not ready: %s", screen())
			return
		}
		ok, _, e := kernel.NewProc("GenerateConsoleCtrlEvent").Call(event, 0)
		if ok == 0 {
			finished <- e
			return
		}
		if !wait(func() bool { return strings.Contains(screen(), "上次命令退出码：23") }) {
			finished <- fmt.Errorf("TUI not restored: %s", screen())
			return
		}
		send("q", 0)
		// The renderer can publish the first frame just before input setup.
		time.Sleep(150 * time.Millisecond)
		send("q", 0)
		finished <- nil
	}()
	code := executeContext(ctx, []string{"-C", project, "tui"}, input, output, output, "test", "")
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	count, err := os.ReadFile(marker + ".done")
	if code != 0 || err != nil || string(count) != "1" {
		t.Fatalf("code=%d count=%s err=%v", code, count, err)
	}
}
