package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func privatePromptConsole(t *testing.T) bool {
	t.Helper()
	if os.Getenv("MYENV_TEST_PRIVATE_PROMPT") != t.Name() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
		cmd.Env = append(os.Environ(), "MYENV_TEST_PRIVATE_PROMPT="+t.Name())
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, HideWindow: true}
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("private console: %v: %s", err, output)
		}
		t.Logf("%s", output)
		return false
	}
	var ids [8]uint32
	count, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids)))
	if count != 1 || ids[0] != uint32(os.Getpid()) {
		t.Fatalf("console not isolated: %d %v %v", count, ids, err)
	}
	return true
}
func TestWindowsPromptReadCancellation(t *testing.T) {
	if !privatePromptConsole(t) {
		return
	}
	input, err := os.Open("CONIN$")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	reader := promptInput(ctx, input)
	if _, ok := reader.(*promptReader); !ok {
		t.Fatal("console input was not wrapped")
	}
	buffer := make([]byte, 64)
	n, err := reader.Read(buffer)
	if n != 0 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("read: %d %v", n, err)
	}
	var mode uint32
	if err := windows.GetConsoleMode(windows.Handle(input.Fd()), &mode); err != nil {
		t.Fatalf("input closed: %v", err)
	}
	t.Log("canceled console read returned; caller input remains open")
}

type interruptPromptWriter struct {
	bytes.Buffer
	trigger   string
	sent      bool
	delivered chan error
}

func (w *interruptPromptWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *interruptPromptWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if !w.sent && bytes.Contains(p, []byte(w.trigger)) {
		w.sent = true
		go func() {
			time.Sleep(100 * time.Millisecond)
			ok, _, e := windows.NewLazySystemDLL("kernel32.dll").NewProc("GenerateConsoleCtrlEvent").Call(0, 0)
			if ok == 0 {
				w.delivered <- e
			} else {
				w.delivered <- nil
			}
		}()
	}
	return n, err
}
func TestWindowsUsePromptInterrupt(t *testing.T)   { testWindowsPromptInterrupt(t, false) }
func TestWindowsInitPromptInterrupt(t *testing.T)  { testWindowsPromptInterrupt(t, false) }
func TestWindowsBuildPromptInterrupt(t *testing.T) { testWindowsPromptInterrupt(t, true) }
func testWindowsPromptInterrupt(t *testing.T, build bool) {
	if !privatePromptConsole(t) {
		return
	}
	ok, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleCtrlHandler").Call(0, 0)
	if ok == 0 {
		t.Fatal(err)
	}
	t.Setenv("CI", "")
	input, err := os.Open("CONIN$")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	root := t.TempDir()
	args := []string{"-C", root, "use"}
	if t.Name() == "TestWindowsInitPromptInterrupt" {
		args = []string{"-C", root, "init"}
	}
	trigger := "Enter tool@version"
	if build {
		args = []string{"-C", root, "sync"}
		trigger = "[y/N]"
		for name, value := range map[string]string{
			"myenv.yaml":          "schema: 1\ntools: {python: '3.12'}\npython: {project: '.'}\n",
			"pyproject.toml":      "[project]\nname='cancel-build'\nversion='0.1.0'\n[build-system]\nrequires=[]\nbuild-backend='must_not_execute'\nbackend-path=['.']\n",
			"must_not_execute.py": "raise RuntimeError('build executed without authorization')\n",
		} {
			if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	diagnostic := &interruptPromptWriter{trigger: trigger, delivered: make(chan error, 1)}
	code := execute(args, input, output, diagnostic, "test", root)
	if !diagnostic.sent {
		t.Fatalf("selection prompt not reached: exit=%d diagnostic=%s", code, diagnostic.String())
	}
	if err := <-diagnostic.delivered; err != nil {
		t.Fatal(err)
	}
	if code != 130 || !strings.Contains(diagnostic.String(), "CANCELED") {
		t.Fatalf("exit=%d: %s", code, diagnostic.String())
	}
	entries, err := os.ReadDir(root)
	if err != nil || (!build && len(entries) != 0) {
		t.Fatalf("cancellation created state: %v %v", entries, err)
	}
	if build {
		if _, err := os.Stat(filepath.Join(root, "myenv.lock")); !os.IsNotExist(err) {
			t.Fatalf("lock created: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, ".myenv", "generations")); !os.IsNotExist(err) {
			t.Fatalf("generation created: %v", err)
		}
	}
	fmt.Fprintln(os.Stdout, "prompt interrupted: exit 130, no installation or declaration change")
}
